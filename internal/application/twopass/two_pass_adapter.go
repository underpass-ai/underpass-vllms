package twopassapp

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type TwoPassAdapter struct {
	settings  Settings
	reasoner  domain.CompletionPort
	formatter domain.CompletionPort
	validator domain.SchemaValidatorPort
	logger    *log.Logger
}

func NewTwoPassAdapter(
	settings Settings,
	reasoner domain.CompletionPort,
	formatter domain.CompletionPort,
	validator domain.SchemaValidatorPort,
	logger *log.Logger,
) *TwoPassAdapter {
	return &TwoPassAdapter{
		settings:  settings,
		reasoner:  reasoner,
		formatter: formatter,
		validator: validator,
		logger:    logger,
	}
}

func (a *TwoPassAdapter) Execute(ctx context.Context, requestID domain.RequestID, request domain.StructuredRequest) (domain.StructuredExecutionResult, *domain.Error) {
	intermediate, pass1Metrics, err := a.runPass1(ctx, requestID, request)
	if err != nil {
		return domain.StructuredExecutionResult{}, err
	}

	output, pass2Metrics, err := a.runPass2(ctx, requestID, request, intermediate)
	if err != nil {
		return domain.StructuredExecutionResult{}, err
	}

	return domain.StructuredExecutionResult{
		IntermediateRepresentation: string(intermediate),
		Output:                     output,
		Metadata: domain.ResponseMetadata{
			ExecutionMode:      domain.ExecutionModeTwoPass,
			Pass1PromptVersion: a.settings.Versions.Pass1Prompt,
			Pass2PromptVersion: a.settings.Versions.Pass2Prompt,
			IRVersion:          a.settings.Versions.IR,
			Pass1:              pass1Metrics,
			Pass2:              pass2Metrics,
		},
	}, nil
}

func (a *TwoPassAdapter) runPass1(ctx context.Context, requestID domain.RequestID, request domain.StructuredRequest) (domain.IntermediateRepresentation, domain.PassMetrics, *domain.Error) {
	options := resolvePassDefaults(a.settings.Pass1, request.Pass1)
	userPrompt := buildPass1Prompt(a.settings.PromptTemplates.Pass1User, request.Input)

	start := time.Now()
	response, err := a.reasoner.Complete(ctx, domain.CompletionRequest{
		Model: options.Model,
		Messages: []domain.Message{
			{Role: domain.SystemRole, Content: options.SystemPrompt},
			{Role: domain.UserRole, Content: userPrompt},
		},
		Temperature:         options.Temperature,
		TopP:                options.TopP,
		TopK:                options.TopK,
		PresencePenalty:     options.PresencePenalty,
		RepetitionPenalty:   options.RepetitionPenalty,
		MaxTokens:           options.MaxTokens,
		ThinkingTokenBudget: options.ThinkingTokenBudget,
		ReasoningEffort:     options.ReasoningEffort,
		PreserveThinking:    options.PreserveThinking,
	})
	latency := time.Since(start)
	if err != nil {
		return "", domain.PassMetrics{}, transportError(domain.ErrorCodePass1Transport, "Pass 1 request failed", err)
	}

	content := strings.TrimSpace(response.Content)
	reasoning := strings.TrimSpace(response.Reasoning)
	usedReasoningFallback := false
	intermediate := domain.IntermediateRepresentation(content)
	if intermediate == "" && reasoning != "" {
		intermediate = domain.IntermediateRepresentation(reasoning)
		usedReasoningFallback = true
	}
	sanitizedIntermediate := domain.IntermediateRepresentation(sanitizeIntermediate(string(intermediate)))
	if sanitizedIntermediate != "" {
		intermediate = sanitizedIntermediate
	}

	metrics := domain.PassMetrics{
		Model:                 options.Model,
		Attempts:              1,
		LatencyMs:             latency.Milliseconds(),
		PromptTokens:          response.Usage.PromptTokens,
		CompletionTokens:      response.Usage.CompletionTokens,
		FinishReason:          response.FinishReason,
		ContentPresent:        content != "",
		ReasoningPresent:      reasoning != "",
		UsedReasoningFallback: usedReasoningFallback,
		Truncated:             response.FinishReason == "length",
	}

	if intermediate == "" {
		return "", metrics, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass1Empty,
			Message:    "Pass 1 returned an empty intermediate representation",
		}
	}
	if err := validateIntermediateSize(intermediate, a.settings.MaxIntermediateBytes); err != nil {
		return "", metrics, err
	}

	a.logger.Printf(
		"request_id=%s adapter=two_pass pass=1 model=%s latency_ms=%d finish_reason=%s content_present=%t reasoning_present=%t used_reasoning_fallback=%t",
		requestID,
		options.Model,
		latency.Milliseconds(),
		response.FinishReason,
		metrics.ContentPresent,
		metrics.ReasoningPresent,
		metrics.UsedReasoningFallback,
	)

	return intermediate, metrics, nil
}

func (a *TwoPassAdapter) runPass2(
	ctx context.Context,
	requestID domain.RequestID,
	request domain.StructuredRequest,
	intermediate domain.IntermediateRepresentation,
) (json.RawMessage, domain.PassMetrics, *domain.Error) {
	options := resolvePassDefaults(a.settings.Pass2, request.Pass2)
	metrics := domain.PassMetrics{Model: options.Model}
	hint := ""

	for attempt := 0; attempt <= a.settings.Pass2RetryCount; attempt++ {
		output, completion, latency, err := a.runPass2Attempt(ctx, requestID, request, intermediate, options, hint)
		metrics.Attempts++
		metrics.LatencyMs += latency.Milliseconds()
		metrics.PromptTokens += completion.Usage.PromptTokens
		metrics.CompletionTokens += completion.Usage.CompletionTokens
		metrics.FinishReason = completion.FinishReason
		metrics.ContentPresent = strings.TrimSpace(completion.Content) != ""
		metrics.ReasoningPresent = strings.TrimSpace(completion.Reasoning) != ""
		metrics.Truncated = completion.FinishReason == "length"
		if err == nil {
			return output, metrics, nil
		}
		if !err.Retryable {
			return nil, metrics, err
		}
		hint = err.Message
	}

	return nil, metrics, &domain.Error{
		StatusCode: 502,
		Code:       domain.ErrorCodePass2Exhausted,
		Message:    "Pass 2 failed after all configured attempts",
	}
}

func (a *TwoPassAdapter) runPass2Attempt(
	ctx context.Context,
	requestID domain.RequestID,
	request domain.StructuredRequest,
	intermediate domain.IntermediateRepresentation,
	options PassDefaults,
	hint string,
) (json.RawMessage, domain.CompletionResponse, time.Duration, *domain.Error) {
	userPrompt := buildPass2Prompt(
		a.settings.PromptTemplates.Pass2User,
		a.settings.PromptTemplates.Pass2RetryHint,
		intermediate,
		request.Schema,
		hint,
	)

	start := time.Now()
	response, err := a.formatter.Complete(ctx, domain.CompletionRequest{
		Model: options.Model,
		Messages: []domain.Message{
			{Role: domain.SystemRole, Content: options.SystemPrompt},
			{Role: domain.UserRole, Content: userPrompt},
		},
		Temperature:         options.Temperature,
		TopP:                options.TopP,
		TopK:                options.TopK,
		PresencePenalty:     options.PresencePenalty,
		RepetitionPenalty:   options.RepetitionPenalty,
		MaxTokens:           options.MaxTokens,
		ThinkingTokenBudget: options.ThinkingTokenBudget,
		ReasoningEffort:     options.ReasoningEffort,
		PreserveThinking:    options.PreserveThinking,
		StructuredSchema:    request.Schema,
		DisableThinking:     true,
	})
	latency := time.Since(start)
	if err != nil {
		return nil, domain.CompletionResponse{}, latency, transportError(domain.ErrorCodePass2Transport, "Pass 2 request failed", err)
	}

	content := strings.TrimSpace(response.Content)
	if content == "" {
		return nil, response, latency, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Empty,
			Message:    "Pass 2 returned an empty body",
			Retryable:  true,
		}
	}

	candidate, err := normalizeJSON(content)
	if err != nil {
		return nil, response, latency, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Validation,
			Message:    err.Error(),
			Retryable:  true,
		}
	}
	if err := a.validator.Validate(request.Schema, candidate); err != nil {
		return nil, response, latency, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Validation,
			Message:    err.Error(),
			Retryable:  true,
		}
	}

	a.logger.Printf(
		"request_id=%s adapter=two_pass pass=2 model=%s latency_ms=%d finish_reason=%s content_present=%t reasoning_present=%t",
		requestID,
		options.Model,
		latency.Milliseconds(),
		response.FinishReason,
		content != "",
		strings.TrimSpace(response.Reasoning) != "",
	)

	return candidate, response, latency, nil
}

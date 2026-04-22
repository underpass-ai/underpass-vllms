package twopassapp

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type SinglePassAdapter struct {
	settings  Settings
	formatter domain.CompletionPort
	logger    *log.Logger
}

func NewSinglePassAdapter(
	settings Settings,
	formatter domain.CompletionPort,
	logger *log.Logger,
) *SinglePassAdapter {
	return &SinglePassAdapter{
		settings:  settings,
		formatter: formatter,
		logger:    logger,
	}
}

func (a *SinglePassAdapter) Execute(ctx context.Context, requestID domain.RequestID, request domain.StructuredRequest) (domain.StructuredExecutionResult, *domain.Error) {
	output, metrics, err := a.runSinglePass(ctx, requestID, request)
	if err != nil {
		return domain.StructuredExecutionResult{}, err
	}

	return domain.StructuredExecutionResult{
		Output:   output,
		Metadata: buildSinglePassMetadata(a.settings, metrics),
	}, nil
}

func (a *SinglePassAdapter) runSinglePass(
	ctx context.Context,
	requestID domain.RequestID,
	request domain.StructuredRequest,
) (json.RawMessage, domain.PassMetrics, *domain.Error) {
	options := resolveSinglePassOptions(a.settings.SinglePass, a.settings.SinglePassSystemPrompt, request.SinglePass, request.Pass2)
	metrics := domain.PassMetrics{Model: options.Model}
	output, completion, latency, err := a.runSinglePassAttempt(ctx, requestID, request, options)
	metrics.Attempts = 1
	metrics.LatencyMs = latency.Milliseconds()
	metrics.PromptTokens = completion.Usage.PromptTokens
	metrics.CompletionTokens = completion.Usage.CompletionTokens
	metrics.FinishReason = completion.FinishReason
	metrics.ContentPresent = strings.TrimSpace(completion.Content) != ""
	metrics.ReasoningPresent = strings.TrimSpace(completion.Reasoning) != ""
	metrics.Truncated = completion.FinishReason == "length"
	if err != nil {
		return nil, metrics, err
	}
	return output, metrics, nil
}

func (a *SinglePassAdapter) runSinglePassAttempt(
	ctx context.Context,
	requestID domain.RequestID,
	request domain.StructuredRequest,
	options PassDefaults,
) (json.RawMessage, domain.CompletionResponse, time.Duration, *domain.Error) {
	start := time.Now()
	response, err := a.formatter.Complete(ctx, buildSinglePassCompletionRequest(a.settings, request, options))
	latency := time.Since(start)
	if err != nil {
		return nil, domain.CompletionResponse{}, latency, transportError(domain.ErrorCodePass2Transport, "Single-pass request failed", err)
	}

	content := strings.TrimSpace(response.Content)
	if content == "" {
		return nil, response, latency, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Empty,
			Message:    "Single-pass request returned an empty body",
		}
	}

	logSinglePassAttempt(a.logger, requestID, options.Model, response, latency)

	return json.RawMessage(content), response, latency, nil
}

func buildSinglePassCompletionRequest(settings Settings, request domain.StructuredRequest, options PassDefaults) domain.CompletionRequest {
	userPrompt := buildSinglePassPrompt(
		settings.PromptTemplates.SinglePassUser,
		settings.PromptTemplates.Pass2RetryHint,
		request.Input,
		request.Schema,
		"",
	)

	return domain.CompletionRequest{
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
	}
}

func logSinglePassAttempt(
	logger *log.Logger,
	requestID domain.RequestID,
	model domain.ModelName,
	response domain.CompletionResponse,
	latency time.Duration,
) {
	if logger == nil {
		return
	}
	logger.Printf(
		"request_id=%s adapter=single_pass model=%s latency_ms=%d finish_reason=%s content_present=%t reasoning_present=%t",
		requestID,
		model,
		latency.Milliseconds(),
		response.FinishReason,
		strings.TrimSpace(response.Content) != "",
		strings.TrimSpace(response.Reasoning) != "",
	)
}

func buildSinglePassMetadata(settings Settings, metrics domain.PassMetrics) domain.ResponseMetadata {
	return domain.ResponseMetadata{
		ExecutionMode:           domain.ExecutionModeSinglePass,
		SinglePassPromptVersion: settings.Versions.SinglePassPrompt,
		IRVersion:               settings.Versions.IR,
		SinglePass:              metrics,
	}
}

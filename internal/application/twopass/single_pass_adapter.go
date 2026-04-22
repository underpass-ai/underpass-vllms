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
		Output: output,
		Metadata: domain.ResponseMetadata{
			ExecutionMode:           domain.ExecutionModeSinglePass,
			SinglePassPromptVersion: a.settings.Versions.SinglePassPrompt,
			IRVersion:               a.settings.Versions.IR,
			SinglePass:              metrics,
		},
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
	userPrompt := buildSinglePassPrompt(
		a.settings.PromptTemplates.SinglePassUser,
		a.settings.PromptTemplates.Pass2RetryHint,
		request.Input,
		request.Schema,
		"",
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
	})
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

	a.logger.Printf(
		"request_id=%s adapter=single_pass model=%s latency_ms=%d finish_reason=%s content_present=%t reasoning_present=%t",
		requestID,
		options.Model,
		latency.Milliseconds(),
		response.FinishReason,
		content != "",
		strings.TrimSpace(response.Reasoning) != "",
	)

	return json.RawMessage(content), response, latency, nil
}

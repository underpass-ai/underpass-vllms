package twopassapp

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type SinglePassStreamAdapter struct {
	settings  Settings
	formatter domain.CompletionStreamPort
	logger    *log.Logger
}

func NewSinglePassStreamAdapter(
	settings Settings,
	formatter domain.CompletionStreamPort,
	logger *log.Logger,
) *SinglePassStreamAdapter {
	return &SinglePassStreamAdapter{
		settings:  settings,
		formatter: formatter,
		logger:    logger,
	}
}

func (a *SinglePassStreamAdapter) Stream(
	ctx context.Context,
	requestID domain.RequestID,
	request domain.StructuredRequest,
	emit func(domain.CompletionDelta) error,
) (domain.StructuredExecutionResult, *domain.Error) {
	options := resolveSinglePassOptions(a.settings.SinglePass, a.settings.SinglePassSystemPrompt, request.SinglePass, request.Pass2)
	metrics := domain.PassMetrics{Model: options.Model}

	start := time.Now()
	response, err := a.formatter.Stream(ctx, buildSinglePassCompletionRequest(a.settings, request, options), emit)
	latency := time.Since(start)
	if err != nil {
		return domain.StructuredExecutionResult{}, transportError(domain.ErrorCodePass2Transport, "Single-pass streaming request failed", err)
	}

	content := strings.TrimSpace(response.Content)
	metrics.Attempts = 1
	metrics.LatencyMs = latency.Milliseconds()
	metrics.PromptTokens = response.Usage.PromptTokens
	metrics.CompletionTokens = response.Usage.CompletionTokens
	metrics.FinishReason = response.FinishReason
	metrics.ContentPresent = content != ""
	metrics.ReasoningPresent = strings.TrimSpace(response.Reasoning) != ""
	metrics.Truncated = response.FinishReason == "length"

	if content == "" {
		return domain.StructuredExecutionResult{}, &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Empty,
			Message:    "Single-pass streaming request returned an empty body",
		}
	}

	logSinglePassAttempt(a.logger, requestID, options.Model, response, latency)

	return domain.StructuredExecutionResult{
		Output:   json.RawMessage(content),
		Metadata: buildSinglePassMetadata(a.settings, metrics),
	}, nil
}

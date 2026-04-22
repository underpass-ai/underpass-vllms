package openaiadapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type Client struct {
	profile    ProviderProfile
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *log.Logger
}

type ProviderProfile string

const (
	ProviderProfileVLLMChatCompletions   ProviderProfile = "vllm_chat_completions"
	ProviderProfileOpenAIChatCompletions ProviderProfile = "openai_chat_completions"
)

type chatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponseDTO struct {
	Choices []chatCompletionChoiceDTO `json:"choices"`
	Usage   usageDTO                  `json:"usage"`
}

type chatCompletionChoiceDTO struct {
	Message      chatCompletionMessageDTO `json:"message"`
	FinishReason string                   `json:"finish_reason"`
}

type chatCompletionMessageDTO struct {
	Content   any `json:"content"`
	Reasoning any `json:"reasoning"`
}

type chatCompletionChunkDTO struct {
	Choices []chatCompletionChunkChoiceDTO `json:"choices"`
	Usage   *usageDTO                      `json:"usage,omitempty"`
}

type chatCompletionChunkChoiceDTO struct {
	Delta        chatCompletionDeltaDTO `json:"delta"`
	FinishReason *string                `json:"finish_reason"`
}

type chatCompletionDeltaDTO struct {
	Content   any `json:"content"`
	Reasoning any `json:"reasoning"`
}

type usageDTO struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func ParseProviderProfile(raw string) (ProviderProfile, error) {
	switch ProviderProfile(strings.TrimSpace(raw)) {
	case ProviderProfileVLLMChatCompletions:
		return ProviderProfileVLLMChatCompletions, nil
	case ProviderProfileOpenAIChatCompletions:
		return ProviderProfileOpenAIChatCompletions, nil
	default:
		return "", fmt.Errorf("unsupported provider profile %q", raw)
	}
}

func NewClient(profile ProviderProfile, baseURL, apiKey string, timeout time.Duration, logger *log.Logger) *Client {
	return &Client{
		profile: profile,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

func (c *Client) Complete(ctx context.Context, request domain.CompletionRequest) (domain.CompletionResponse, error) {
	payload := c.buildPayload(request)

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		return domain.CompletionResponse{}, fmt.Errorf("unexpected status %d: %s", httpResponse.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded chatCompletionResponseDTO
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		c.logDiagnostic("empty_choices", request.Model, nil, nil, "", decoded.Usage, responseBody)
		return domain.CompletionResponse{}, nil
	}

	content, reasoning, err := extractMessageParts(decoded.Choices[0].Message)
	if err != nil {
		return domain.CompletionResponse{}, err
	}
	if strings.TrimSpace(content) == "" {
		choice := decoded.Choices[0]
		c.logDiagnostic("empty_content", request.Model, choice.Message.Content, choice.Message.Reasoning, choice.FinishReason, decoded.Usage, responseBody)
	}

	return domain.CompletionResponse{
		Content:      content,
		Reasoning:    reasoning,
		FinishReason: decoded.Choices[0].FinishReason,
		Usage: domain.Usage{
			PromptTokens:     decoded.Usage.PromptTokens,
			CompletionTokens: decoded.Usage.CompletionTokens,
			TotalTokens:      decoded.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) Stream(
	ctx context.Context,
	request domain.CompletionRequest,
	emit func(domain.CompletionDelta) error,
) (domain.CompletionResponse, error) {
	payload := c.buildPayload(request)
	payload["stream"] = true

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(httpResponse.Body)
		if readErr != nil {
			return domain.CompletionResponse{}, fmt.Errorf("read error response: %w", readErr)
		}
		return domain.CompletionResponse{}, fmt.Errorf("unexpected status %d: %s", httpResponse.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	usage := domain.Usage{}
	finishReason := ""

	scanner := bufio.NewScanner(httpResponse.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payloadLine := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payloadLine == "[DONE]" {
			break
		}

		var chunk chatCompletionChunkDTO
		if err := json.Unmarshal([]byte(payloadLine), &chunk); err != nil {
			return domain.CompletionResponse{}, fmt.Errorf("decode stream chunk: %w", err)
		}
		if chunk.Usage != nil {
			usage = domain.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		content, reasoning, err := extractChunkDeltaParts(choice.Delta)
		if err != nil {
			return domain.CompletionResponse{}, err
		}
		if content != "" {
			contentBuilder.WriteString(content)
			if emit != nil {
				if err := emit(domain.CompletionDelta{Content: content}); err != nil {
					return domain.CompletionResponse{}, err
				}
			}
		}
		if reasoning != "" {
			reasoningBuilder.WriteString(reasoning)
			if emit != nil {
				if err := emit(domain.CompletionDelta{Reasoning: reasoning}); err != nil {
					return domain.CompletionResponse{}, err
				}
			}
		}
		if choice.FinishReason != nil {
			finishReason = *choice.FinishReason
		}
	}

	if err := scanner.Err(); err != nil {
		return domain.CompletionResponse{}, fmt.Errorf("read stream: %w", err)
	}

	return domain.CompletionResponse{
		Content:      contentBuilder.String(),
		Reasoning:    reasoningBuilder.String(),
		FinishReason: finishReason,
		Usage:        usage,
	}, nil
}

func (c *Client) logDiagnostic(
	event string,
	model domain.ModelName,
	content any,
	reasoning any,
	finishReason string,
	usage usageDTO,
	responseBody []byte,
) {
	if c.logger == nil {
		return
	}
	c.logger.Printf(
		"openai_client_diagnostic event=%s profile=%s model=%s finish_reason=%s prompt_tokens=%d completion_tokens=%d total_tokens=%d content_type=%s reasoning_type=%s body=%q",
		event,
		c.profile,
		model,
		finishReason,
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
		describeType(content),
		describeType(reasoning),
		truncateForLog(responseBody, 4096),
	)
}

func describeType(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}

func truncateForLog(value []byte, limit int) string {
	if len(value) <= limit {
		return string(value)
	}
	return string(value[:limit]) + "...<truncated>"
}

func (c *Client) buildPayload(request domain.CompletionRequest) map[string]any {
	messages := make([]chatMessageDTO, 0, len(request.Messages))
	for _, message := range request.Messages {
		role := string(message.Role)
		if c.profile == ProviderProfileOpenAIChatCompletions && message.Role == domain.SystemRole {
			role = "developer"
		}
		messages = append(messages, chatMessageDTO{
			Role:    role,
			Content: message.Content,
		})
	}

	switch c.profile {
	case ProviderProfileOpenAIChatCompletions:
		return buildOpenAIChatPayload(request, messages)
	default:
		return buildVLLMChatPayload(request, messages)
	}
}

func buildVLLMChatPayload(request domain.CompletionRequest, messages []chatMessageDTO) map[string]any {
	payload := map[string]any{
		"model":       request.Model,
		"messages":    messages,
		"temperature": request.Temperature,
		"max_tokens":  request.MaxTokens,
	}
	if request.TopP != nil {
		payload["top_p"] = *request.TopP
	}
	if request.TopK != nil {
		payload["top_k"] = *request.TopK
	}
	if request.PresencePenalty != nil {
		payload["presence_penalty"] = *request.PresencePenalty
	}
	if request.RepetitionPenalty != nil {
		payload["repetition_penalty"] = *request.RepetitionPenalty
	}
	if request.ThinkingTokenBudget != nil {
		payload["thinking_token_budget"] = *request.ThinkingTokenBudget
	}
	if len(request.StructuredSchema) > 0 {
		payload["structured_outputs"] = map[string]any{
			"json": json.RawMessage(request.StructuredSchema),
		}
	}
	chatTemplateKwargs := map[string]any{}
	if request.DisableThinking {
		chatTemplateKwargs["enable_thinking"] = false
	}
	if request.PreserveThinking != nil {
		chatTemplateKwargs["preserve_thinking"] = *request.PreserveThinking
	}
	if len(chatTemplateKwargs) > 0 {
		payload["chat_template_kwargs"] = chatTemplateKwargs
	}
	return payload
}

func buildOpenAIChatPayload(request domain.CompletionRequest, messages []chatMessageDTO) map[string]any {
	payload := map[string]any{
		"model":                 request.Model,
		"messages":              messages,
		"temperature":           request.Temperature,
		"max_completion_tokens": request.MaxTokens,
	}
	if request.TopP != nil {
		payload["top_p"] = *request.TopP
	}
	if request.PresencePenalty != nil {
		payload["presence_penalty"] = *request.PresencePenalty
	}
	if request.ReasoningEffort != nil {
		payload["reasoning_effort"] = *request.ReasoningEffort
	}
	if len(request.StructuredSchema) > 0 {
		payload["response_format"] = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "structured_output",
				"strict": true,
				"schema": json.RawMessage(request.StructuredSchema),
			},
		}
	}
	return payload
}

func extractMessageParts(message chatCompletionMessageDTO) (string, string, error) {
	content, err := extractContent(message.Content)
	if err != nil {
		return "", "", err
	}
	reasoning, err := extractContent(message.Reasoning)
	if err != nil {
		return "", "", err
	}
	return content, reasoning, nil
}

func extractChunkDeltaParts(delta chatCompletionDeltaDTO) (string, string, error) {
	content, err := extractContent(delta.Content)
	if err != nil {
		return "", "", err
	}
	reasoning, err := extractContent(delta.Reasoning)
	if err != nil {
		return "", "", err
	}
	return content, reasoning, nil
}

func extractContent(raw any) (string, error) {
	switch value := raw.(type) {
	case nil:
		return "", nil
	case string:
		return value, nil
	case []any:
		var builder strings.Builder
		for _, item := range value {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok {
				builder.WriteString(text)
			}
		}
		return builder.String(), nil
	default:
		return "", fmt.Errorf("unsupported content type %T", value)
	}
}

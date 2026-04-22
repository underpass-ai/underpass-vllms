package openaiadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestExtractMessagePartsReturnsContentAndReasoning(t *testing.T) {
	message := chatCompletionMessageDTO{
		Content:   "final ir",
		Reasoning: "internal reasoning",
	}

	content, reasoning, err := extractMessageParts(message)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "final ir" {
		t.Fatalf("expected content, got %q", content)
	}
	if reasoning != "internal reasoning" {
		t.Fatalf("expected reasoning, got %q", reasoning)
	}
}

func TestExtractMessagePartsSupportsTokenizedReasoning(t *testing.T) {
	message := chatCompletionMessageDTO{
		Content: nil,
		Reasoning: []any{
			map[string]any{"text": "first "},
			map[string]any{"text": "second"},
		},
	}

	content, reasoning, err := extractMessageParts(message)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content, got %q", content)
	}
	if reasoning != "first second" {
		t.Fatalf("expected concatenated reasoning, got %q", reasoning)
	}
}

func TestCompleteIncludesSamplingAndThinkingControls(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"choices":[{"message":{"content":"ok","reasoning":"thought"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	topP := 0.95
	topK := 20
	presencePenalty := 0.0
	repetitionPenalty := 1.0
	thinkingTokenBudget := 2048
	preserveThinking := true

	client := NewClient(ProviderProfileVLLMChatCompletions, server.URL, "EMPTY", time.Second, nil)
	_, err := client.Complete(context.Background(), domain.CompletionRequest{
		Model:               "Qwen/Qwen3.6-35B-A3B",
		Messages:            []domain.Message{{Role: domain.UserRole, Content: "hello"}},
		Temperature:         0.6,
		TopP:                &topP,
		TopK:                &topK,
		PresencePenalty:     &presencePenalty,
		RepetitionPenalty:   &repetitionPenalty,
		MaxTokens:           4096,
		ThinkingTokenBudget: &thinkingTokenBudget,
		PreserveThinking:    &preserveThinking,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["top_p"] != topP {
		t.Fatalf("expected top_p=%v, got %#v", topP, got["top_p"])
	}
	if got["top_k"] != float64(topK) {
		t.Fatalf("expected top_k=%d, got %#v", topK, got["top_k"])
	}
	if got["presence_penalty"] != presencePenalty {
		t.Fatalf("expected presence_penalty=%v, got %#v", presencePenalty, got["presence_penalty"])
	}
	if got["repetition_penalty"] != repetitionPenalty {
		t.Fatalf("expected repetition_penalty=%v, got %#v", repetitionPenalty, got["repetition_penalty"])
	}
	if got["thinking_token_budget"] != float64(thinkingTokenBudget) {
		t.Fatalf("expected thinking_token_budget=%d, got %#v", thinkingTokenBudget, got["thinking_token_budget"])
	}
	chatTemplateKwargs, ok := got["chat_template_kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("expected chat_template_kwargs map, got %#v", got["chat_template_kwargs"])
	}
	if chatTemplateKwargs["preserve_thinking"] != preserveThinking {
		t.Fatalf("expected preserve_thinking=%t, got %#v", preserveThinking, chatTemplateKwargs["preserve_thinking"])
	}
}

func TestCompleteBuildsOpenAIChatStructuredPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"choices":[{"message":{"content":"{\"kind\":\"bug\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	defer server.Close()

	topP := 0.95
	presencePenalty := 0.0
	reasoningEffort := domain.ReasoningEffort("high")

	client := NewClient(ProviderProfileOpenAIChatCompletions, server.URL, "EMPTY", time.Second, nil)
	_, err := client.Complete(context.Background(), domain.CompletionRequest{
		Model:            "gpt-oss-20b",
		Messages:         []domain.Message{{Role: domain.SystemRole, Content: "be precise"}, {Role: domain.UserRole, Content: "hello"}},
		Temperature:      0.2,
		TopP:             &topP,
		PresencePenalty:  &presencePenalty,
		MaxTokens:        2048,
		ReasoningEffort:  &reasoningEffort,
		StructuredSchema: json.RawMessage(`{"type":"object","properties":{"kind":{"type":"string"}},"required":["kind"],"additionalProperties":false}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["max_completion_tokens"] != float64(2048) {
		t.Fatalf("expected max_completion_tokens=2048, got %#v", got["max_completion_tokens"])
	}
	if _, exists := got["max_tokens"]; exists {
		t.Fatalf("did not expect max_tokens in OpenAI payload")
	}
	if got["reasoning_effort"] != "high" {
		t.Fatalf("expected reasoning_effort=high, got %#v", got["reasoning_effort"])
	}
	messages, ok := got["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("expected two messages, got %#v", got["messages"])
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first message map, got %#v", messages[0])
	}
	if firstMessage["role"] != "developer" {
		t.Fatalf("expected system role to map to developer, got %#v", firstMessage["role"])
	}
	if _, exists := got["structured_outputs"]; exists {
		t.Fatalf("did not expect structured_outputs in OpenAI payload")
	}
	responseFormat, ok := got["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("expected response_format map, got %#v", got["response_format"])
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("expected response_format.type=json_schema, got %#v", responseFormat["type"])
	}
	jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("expected json_schema map, got %#v", responseFormat["json_schema"])
	}
	if jsonSchema["name"] != "structured_output" {
		t.Fatalf("expected json_schema.name=structured_output, got %#v", jsonSchema["name"])
	}
	if jsonSchema["strict"] != true {
		t.Fatalf("expected json_schema.strict=true, got %#v", jsonSchema["strict"])
	}
}

func TestStreamAggregatesContentAndReasoning(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(strings.Join([]string{
			`data: {"choices":[{"delta":{"content":"{\"value\":"},"finish_reason":null}]}`,
			``,
			`data: {"choices":[{"delta":{"reasoning":"hidden"},"finish_reason":null}]}`,
			``,
			`data: {"choices":[{"delta":{"content":"\"hello\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`,
			``,
			`data: [DONE]`,
			``,
		}, "\n")))
	}))
	defer server.Close()

	client := NewClient(ProviderProfileVLLMChatCompletions, server.URL, "EMPTY", time.Second, nil)
	var seen []domain.CompletionDelta
	response, err := client.Stream(context.Background(), domain.CompletionRequest{
		Model:     "google/gemma-4-31B-it",
		Messages:  []domain.Message{{Role: domain.UserRole, Content: "hello"}},
		MaxTokens: 256,
	}, func(delta domain.CompletionDelta) error {
		seen = append(seen, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["stream"] != true {
		t.Fatalf("expected stream=true, got %#v", got["stream"])
	}
	if response.Content != `{"value":"hello"}` {
		t.Fatalf("unexpected content: %q", response.Content)
	}
	if response.Reasoning != "hidden" {
		t.Fatalf("unexpected reasoning: %q", response.Reasoning)
	}
	if response.FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: %q", response.FinishReason)
	}
	if response.Usage.TotalTokens != 18 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
	if len(seen) != 3 {
		t.Fatalf("expected 3 deltas, got %d", len(seen))
	}
	if seen[0].Content != "{\"value\":" || seen[1].Reasoning != "hidden" || seen[2].Content != "\"hello\"}" {
		t.Fatalf("unexpected deltas: %#v", seen)
	}
}

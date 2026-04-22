package openaiadapter

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestParseProviderProfile(t *testing.T) {
	if _, err := ParseProviderProfile("vllm_chat_completions"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := ParseProviderProfile(" openai_chat_completions "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := ParseProviderProfile("bad"); err == nil {
		t.Fatalf("expected invalid profile error")
	}
}

func TestCompleteLogsEmptyChoices(t *testing.T) {
	var buffer bytes.Buffer
	logger := log.New(&buffer, "", 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()

	client := NewClient(ProviderProfileVLLMChatCompletions, server.URL, "EMPTY", time.Second, logger)
	response, err := client.Complete(context.Background(), domain.CompletionRequest{
		Model:     "google/gemma-4-31B-it",
		Messages:  []domain.Message{{Role: domain.UserRole, Content: "hello"}},
		MaxTokens: 64,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Content != "" {
		t.Fatalf("expected empty response content, got %q", response.Content)
	}
	if !strings.Contains(buffer.String(), "event=empty_choices") {
		t.Fatalf("expected diagnostic log, got %q", buffer.String())
	}
}

func TestUtilityHelpers(t *testing.T) {
	if got := describeType(nil); got != "<nil>" {
		t.Fatalf("unexpected nil type: %q", got)
	}
	if got := truncateForLog([]byte("abcdef"), 3); got != "abc...<truncated>" {
		t.Fatalf("unexpected truncation: %q", got)
	}
}

func TestLogDiagnosticWritesContentAndReasoningTypes(t *testing.T) {
	var buffer bytes.Buffer
	client := &Client{profile: ProviderProfileVLLMChatCompletions, logger: log.New(&buffer, "", 0)}

	client.logDiagnostic("empty_content", "model", "text", []any{map[string]any{"text": "r"}}, "length", usageDTO{
		PromptTokens:     1,
		CompletionTokens: 2,
		TotalTokens:      3,
	}, []byte(`{"choices":[]}`))

	logLine := buffer.String()
	if !strings.Contains(logLine, "content_type=string") || !strings.Contains(logLine, "reasoning_type=[]interface {}") {
		t.Fatalf("unexpected log line: %q", logLine)
	}
}

func TestCompleteReturnsErrorOnUnsupportedMessageParts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":{"bad":"type"}},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer server.Close()

	client := NewClient(ProviderProfileVLLMChatCompletions, server.URL, "EMPTY", time.Second, nil)
	_, err := client.Complete(context.Background(), domain.CompletionRequest{
		Model:     "google/gemma-4-31B-it",
		Messages:  []domain.Message{{Role: domain.UserRole, Content: "hello"}},
		MaxTokens: 64,
	})
	if err == nil {
		t.Fatalf("expected unsupported content type error")
	}
}

func TestStreamReturnsErrorOnBadChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {bad json}\n\n"))
	}))
	defer server.Close()

	client := NewClient(ProviderProfileVLLMChatCompletions, server.URL, "EMPTY", time.Second, nil)
	_, err := client.Stream(context.Background(), domain.CompletionRequest{
		Model:     "google/gemma-4-31B-it",
		Messages:  []domain.Message{{Role: domain.UserRole, Content: "hello"}},
		MaxTokens: 64,
	}, func(delta domain.CompletionDelta) error { return nil })
	if err == nil {
		t.Fatalf("expected bad chunk error")
	}
}

func TestBuildPayloadUsesDeveloperRoleForOpenAIProfile(t *testing.T) {
	client := &Client{profile: ProviderProfileOpenAIChatCompletions}
	payload := client.buildPayload(domain.CompletionRequest{
		Model:     "gpt-oss-20b",
		Messages:  []domain.Message{{Role: domain.SystemRole, Content: "system"}, {Role: domain.UserRole, Content: "user"}},
		MaxTokens: 32,
	})

	rawMessages, ok := payload["messages"].([]chatMessageDTO)
	if !ok {
		t.Fatalf("unexpected messages payload: %#v", payload["messages"])
	}
	if rawMessages[0].Role != "developer" {
		t.Fatalf("expected developer role, got %q", rawMessages[0].Role)
	}
}

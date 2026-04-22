package twopassapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type completionStreamStub struct {
	response domain.CompletionResponse
	err      error
	request  domain.CompletionRequest
	deltas   []domain.CompletionDelta
	calls    int
}

func (s *completionStreamStub) Stream(_ context.Context, request domain.CompletionRequest, emit func(domain.CompletionDelta) error) (domain.CompletionResponse, error) {
	s.request = request
	s.calls++
	for _, delta := range s.deltas {
		if err := emit(delta); err != nil {
			return domain.CompletionResponse{}, err
		}
	}
	return s.response, s.err
}

type structuredStreamExecutionStub struct {
	result  domain.StructuredExecutionResult
	err     *domain.Error
	request domain.StructuredRequest
	calls   int
}

func (s *structuredStreamExecutionStub) Stream(_ context.Context, requestID domain.RequestID, request domain.StructuredRequest, emit func(domain.CompletionDelta) error) (domain.StructuredExecutionResult, *domain.Error) {
	s.calls++
	s.request = request
	if requestID == "" {
		return domain.StructuredExecutionResult{}, &domain.Error{StatusCode: 500, Message: "missing request id"}
	}
	if emit != nil {
		_ = emit(domain.CompletionDelta{Content: "ignored"})
	}
	return s.result, s.err
}

func TestSinglePassStreamAdapterStreamSuccess(t *testing.T) {
	streamer := &completionStreamStub{
		response: domain.CompletionResponse{
			Content:      `{"value":"hello"}`,
			FinishReason: "stop",
			Usage:        domain.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		},
		deltas: []domain.CompletionDelta{
			{Content: `{"value":`},
			{Content: `"hello"}`},
		},
	}
	logger := log.New(io.Discard, "", 0)
	adapter := NewSinglePassStreamAdapter(
		Settings{
			SinglePassSystemPrompt: "single-pass system",
			Versions: Versions{
				SinglePassPrompt: "sp1",
				IR:               "1.0.0",
			},
			PromptTemplates: PromptTemplates{
				Pass2RetryHint: "hint={{hint}}",
				SinglePassUser: "schema={{schema}} input={{input}}",
			},
			SinglePass: PassDefaults{Model: "google/gemma-4-31B-it", Temperature: 0.2, MaxTokens: 256},
		},
		streamer,
		logger,
	)

	var emitted []domain.CompletionDelta
	result, execErr := adapter.Stream(context.Background(), "req-stream", domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	}, func(delta domain.CompletionDelta) error {
		emitted = append(emitted, delta)
		return nil
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if streamer.calls != 1 {
		t.Fatalf("expected a single stream call, got %d", streamer.calls)
	}
	if len(emitted) != 2 {
		t.Fatalf("expected emitted deltas, got %d", len(emitted))
	}
	if string(result.Output) != `{"value":"hello"}` {
		t.Fatalf("unexpected output: %s", string(result.Output))
	}
	if result.Metadata.ExecutionMode != domain.ExecutionModeSinglePass {
		t.Fatalf("unexpected execution mode: %q", result.Metadata.ExecutionMode)
	}
	if result.Metadata.SinglePass.FinishReason != "stop" {
		t.Fatalf("unexpected finish reason: %q", result.Metadata.SinglePass.FinishReason)
	}
}

func TestSinglePassStreamAdapterStreamFailsOnEmptyContent(t *testing.T) {
	streamer := &completionStreamStub{
		response: domain.CompletionResponse{Content: "", FinishReason: "stop"},
	}
	adapter := NewSinglePassStreamAdapter(
		Settings{
			SinglePassSystemPrompt: "single-pass system",
			Versions:               Versions{SinglePassPrompt: "sp1", IR: "1.0.0"},
			PromptTemplates:        PromptTemplates{Pass2RetryHint: "hint={{hint}}", SinglePassUser: "schema={{schema}} input={{input}}"},
			SinglePass:             PassDefaults{Model: "google/gemma-4-31B-it", Temperature: 0.2, MaxTokens: 256},
		},
		streamer,
		log.New(io.Discard, "", 0),
	)

	_, execErr := adapter.Stream(context.Background(), "req-stream", domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	}, func(delta domain.CompletionDelta) error { return nil })
	if execErr == nil {
		t.Fatalf("expected empty-content error")
	}
	if execErr.Code != domain.ErrorCodePass2Empty {
		t.Fatalf("unexpected error code: %q", execErr.Code)
	}
}

func TestSinglePassStreamAdapterWrapsTransportFailure(t *testing.T) {
	streamer := &completionStreamStub{err: errors.New("boom")}
	adapter := NewSinglePassStreamAdapter(
		Settings{
			SinglePassSystemPrompt: "single-pass system",
			Versions:               Versions{SinglePassPrompt: "sp1", IR: "1.0.0"},
			PromptTemplates:        PromptTemplates{Pass2RetryHint: "hint={{hint}}", SinglePassUser: "schema={{schema}} input={{input}}"},
			SinglePass:             PassDefaults{Model: "google/gemma-4-31B-it", Temperature: 0.2, MaxTokens: 256},
		},
		streamer,
		log.New(io.Discard, "", 0),
	)

	_, execErr := adapter.Stream(context.Background(), "req-stream", domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	}, func(delta domain.CompletionDelta) error { return nil })
	if execErr == nil {
		t.Fatalf("expected transport error")
	}
	if execErr.Code != domain.ErrorCodePass2Transport {
		t.Fatalf("unexpected error code: %q", execErr.Code)
	}
}

func TestStreamServiceAssignsRequestIDAndSchemaVersion(t *testing.T) {
	service := NewStreamService(&structuredStreamExecutionStub{
		result: domain.StructuredExecutionResult{
			Output:   json.RawMessage(`{"ok":true}`),
			Metadata: domain.ResponseMetadata{ExecutionMode: domain.ExecutionModeSinglePass},
		},
	})

	response, execErr := service.Stream(context.Background(), domain.StructuredRequest{
		Input:         "hello",
		SchemaVersion: "v1",
		Schema:        json.RawMessage(`{"type":"object"}`),
	}, func(delta domain.CompletionDelta) error { return nil })
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if response.RequestID == "" {
		t.Fatalf("expected request id to be generated")
	}
	if response.Metadata.SchemaVersion != "v1" {
		t.Fatalf("unexpected schema version: %q", response.Metadata.SchemaVersion)
	}
}

func TestStreamServiceValidatesRequest(t *testing.T) {
	service := NewStreamService(&structuredStreamExecutionStub{})

	_, execErr := service.Stream(context.Background(), domain.StructuredRequest{
		Input:  "",
		Schema: json.RawMessage(`{"type":"object"}`),
	}, func(delta domain.CompletionDelta) error { return nil })
	if execErr == nil {
		t.Fatalf("expected validation error")
	}
	if execErr.Code != domain.ErrorCodeInvalidRequest {
		t.Fatalf("unexpected error code: %q", execErr.Code)
	}
}

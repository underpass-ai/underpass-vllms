package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type useCaseStub struct {
	response domain.StructuredResponse
	err      *domain.Error
	request  domain.StructuredRequest
}

func (s *useCaseStub) Execute(_ context.Context, request domain.StructuredRequest) (domain.StructuredResponse, *domain.Error) {
	s.request = request
	return s.response, s.err
}

func TestStructuredReturnsUseCaseResponse(t *testing.T) {
	handler := NewHandler(&useCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-1",
			Output:    json.RawMessage(`{"ok":true}`),
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/two-pass/structured", strings.NewReader(`{"input":"hello","schema":{"type":"object"}}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"request_id":"req-1"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestChatCompletionsReturnsOpenAICompatibleResponse(t *testing.T) {
	stub := &useCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-2",
			Output:    json.RawMessage(`{"value":"hello"}`),
			Metadata: domain.ResponseMetadata{
				ExecutionMode:           domain.ExecutionModeSinglePass,
				SinglePassPromptVersion: "2026-04-21.1",
				IRVersion:               "1.0.0",
				SinglePass: domain.PassMetrics{
					Model:            "google/gemma-4-31B-it",
					Attempts:         1,
					LatencyMs:        321,
					PromptTokens:     12,
					CompletionTokens: 7,
					FinishReason:     "stop",
					ContentPresent:   true,
				},
			},
		},
	}

	handler := NewHandler(stub, WithPublicModel("google/gemma-4-31B-it"))

	body := `{
		"model":"google/gemma-4-31B-it",
		"messages":[
			{"role":"developer","content":"Extract strictly"},
			{"role":"user","content":"Return hello in the value field"}
		],
		"response_format":{
			"type":"json_schema",
			"json_schema":{
				"name":"hello_schema",
				"schema":{
					"type":"object",
					"properties":{"value":{"type":"string"}},
					"required":["value"],
					"additionalProperties":false
				},
				"strict":true
			}
		},
		"temperature":0.2,
		"max_completion_tokens":128
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"object":"chat.completion"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"content":"{\"value\":\"hello\"}"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"model":"google/gemma-4-31B-it"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}

	if stub.request.SchemaVersion != "hello_schema" {
		t.Fatalf("unexpected schema version: %q", stub.request.SchemaVersion)
	}
	if stub.request.IncludeIntermediate == nil || *stub.request.IncludeIntermediate {
		t.Fatalf("expected include_intermediate=false for chat completions")
	}
	if stub.request.SinglePass == nil || stub.request.Pass1 == nil || stub.request.Pass2 == nil {
		t.Fatalf("expected model overrides to be propagated to all execution paths")
	}
	if stub.request.SinglePass.MaxTokens == nil || *stub.request.SinglePass.MaxTokens != 128 {
		t.Fatalf("unexpected max tokens override: %#v", stub.request.SinglePass.MaxTokens)
	}
	if !strings.Contains(string(stub.request.Input), "DEVELOPER:\nExtract strictly") {
		t.Fatalf("expected developer message in mapped input, got %q", stub.request.Input)
	}
	if !strings.Contains(string(stub.request.Input), "USER:\nReturn hello in the value field") {
		t.Fatalf("expected user message in mapped input, got %q", stub.request.Input)
	}
}

func TestChatCompletionsRejectsMissingResponseFormat(t *testing.T) {
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"))

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"google/gemma-4-31B-it","messages":[{"role":"user","content":"hello"}]}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"invalid_request_error"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestModelsReturnsConfiguredPublicModel(t *testing.T) {
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"))

	request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"id":"google/gemma-4-31B-it"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

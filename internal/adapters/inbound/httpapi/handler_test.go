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

type streamUseCaseStub struct {
	response domain.StructuredResponse
	err      *domain.Error
	request  domain.StructuredRequest
	deltas   []domain.CompletionDelta
}

func (s *streamUseCaseStub) Stream(_ context.Context, request domain.StructuredRequest, emit func(domain.CompletionDelta) error) (domain.StructuredResponse, *domain.Error) {
	s.request = request
	for _, delta := range s.deltas {
		if err := emit(delta); err != nil {
			return domain.StructuredResponse{}, &domain.Error{
				StatusCode: 500,
				Code:       domain.ErrorCodePass2Transport,
				Message:    err.Error(),
			}
		}
	}
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

func TestChatCompletionsStreamsOpenAICompatibleChunks(t *testing.T) {
	streamStub := &streamUseCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-stream-chat",
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
		deltas: []domain.CompletionDelta{
			{Content: `{"value":`},
			{Content: `"hello"}`},
		},
	}
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"), WithStreamer(streamStub))

	body := `{
		"model":"google/gemma-4-31B-it",
		"messages":[{"role":"user","content":"Return hello in the value field"}],
		"response_format":{
			"type":"json_schema",
			"json_schema":{
				"name":"hello_schema",
				"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}
			}
		},
		"stream":true
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), `"object":"chat.completion.chunk"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"role":"assistant"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"content":"{\"value\":"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"finish_reason":"stop"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `data: [DONE]`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if streamStub.request.RequestID == "" {
		t.Fatalf("expected request_id to be injected for streaming")
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

func TestChatCompletionsRejectsStreamingWithoutSinglePassStreamer(t *testing.T) {
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"))

	body := `{
		"model":"google/gemma-4-31B-it",
		"messages":[{"role":"user","content":"Return hello in the value field"}],
		"response_format":{
			"type":"json_schema",
			"json_schema":{
				"name":"hello_schema",
				"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}
			}
		},
		"stream":true
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `stream=true is only supported for single_pass backends`) {
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

func TestResponsesReturnsOpenAICompatibleResponse(t *testing.T) {
	stub := &useCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-3",
			Output:    json.RawMessage(`{"value":"hello"}`),
			Metadata: domain.ResponseMetadata{
				ExecutionMode:           domain.ExecutionModeSinglePass,
				SinglePassPromptVersion: "2026-04-21.1",
				IRVersion:               "1.0.0",
				SinglePass: domain.PassMetrics{
					Model:            "google/gemma-4-31B-it",
					Attempts:         1,
					LatencyMs:        222,
					PromptTokens:     10,
					CompletionTokens: 5,
					FinishReason:     "stop",
					ContentPresent:   true,
				},
			},
		},
	}

	handler := NewHandler(stub, WithPublicModel("google/gemma-4-31B-it"))

	body := `{
		"model":"google/gemma-4-31B-it",
		"instructions":"Extract strictly",
		"input":"Return hello in the value field",
		"text":{
			"format":{
				"type":"json_schema",
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
		"max_output_tokens":64,
		"reasoning":{"effort":"low"}
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"object":"response"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"output_text":"{\"value\":\"hello\"}"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"output_text"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}

	if stub.request.SchemaVersion != "hello_schema" {
		t.Fatalf("unexpected schema version: %q", stub.request.SchemaVersion)
	}
	if !strings.Contains(string(stub.request.Input), "DEVELOPER:\nExtract strictly") {
		t.Fatalf("expected instructions in mapped input, got %q", stub.request.Input)
	}
	if !strings.Contains(string(stub.request.Input), "USER:\nReturn hello in the value field") {
		t.Fatalf("expected input in mapped input, got %q", stub.request.Input)
	}
	if stub.request.SinglePass == nil || stub.request.SinglePass.MaxTokens == nil || *stub.request.SinglePass.MaxTokens != 64 {
		t.Fatalf("unexpected max_output_tokens mapping: %#v", stub.request.SinglePass)
	}
}

func TestResponsesStreamsSemanticEvents(t *testing.T) {
	streamStub := &streamUseCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-stream-response",
			Output:    json.RawMessage(`{"value":"hello"}`),
			Metadata: domain.ResponseMetadata{
				ExecutionMode:           domain.ExecutionModeSinglePass,
				SinglePassPromptVersion: "2026-04-21.1",
				IRVersion:               "1.0.0",
				SinglePass: domain.PassMetrics{
					Model:            "google/gemma-4-31B-it",
					Attempts:         1,
					LatencyMs:        222,
					PromptTokens:     10,
					CompletionTokens: 5,
					FinishReason:     "stop",
					ContentPresent:   true,
				},
			},
		},
		deltas: []domain.CompletionDelta{
			{Content: `{"value":`},
			{Content: `"hello"}`},
		},
	}
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"), WithStreamer(streamStub))

	body := `{
		"model":"google/gemma-4-31B-it",
		"input":"Return hello in the value field",
		"text":{
			"format":{
				"type":"json_schema",
				"name":"hello_schema",
				"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}
			}
		},
		"stream":true
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), `"type":"response.created"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"response.output_text.delta"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"response.output_text.done"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"response.completed"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), `[DONE]`) {
		t.Fatalf("responses streaming must not emit [DONE]: %s", recorder.Body.String())
	}
	if streamStub.request.RequestID == "" {
		t.Fatalf("expected request_id to be injected for streaming")
	}
}

func TestResponsesRejectsTextFormat(t *testing.T) {
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"))

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/responses",
		strings.NewReader(`{"model":"google/gemma-4-31B-it","input":"hello","text":{"format":{"type":"text"}}}`),
	)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `text.format.type=text is not supported`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestChatCompletionsMapsDomainServerErrorToOpenAIErrorShape(t *testing.T) {
	handler := NewHandler(&useCaseStub{
		err: &domain.Error{
			StatusCode: 502,
			Code:       domain.ErrorCodePass2Transport,
			Message:    "Pass 2 request failed",
			Retryable:  true,
		},
	}, WithPublicModel("google/gemma-4-31B-it"))

	body := `{
		"model":"google/gemma-4-31B-it",
		"messages":[{"role":"user","content":"hello"}],
		"response_format":{
			"type":"json_schema",
			"json_schema":{
				"name":"hello_schema",
				"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}
			}
		}
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"server_error"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"pass2_transport_failure"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestResponsesMapsDomainInvalidRequestToOpenAIErrorShape(t *testing.T) {
	handler := NewHandler(&useCaseStub{
		err: &domain.Error{
			StatusCode: 400,
			Code:       domain.ErrorCodeInvalidRequest,
			Message:    "input is required",
		},
	}, WithPublicModel("google/gemma-4-31B-it"))

	body := `{
		"model":"google/gemma-4-31B-it",
		"input":"hello",
		"text":{
			"format":{
				"type":"json_schema",
				"name":"hello_schema",
				"schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}
			}
		}
	}`
	request := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"invalid_request_error"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"param":"input"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

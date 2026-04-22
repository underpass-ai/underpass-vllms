package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestHealthzAndModelsAndModelByID(t *testing.T) {
	handler := NewHandler(&useCaseStub{}, WithPublicModel("google/gemma-4-31B-it"))

	for _, path := range []string{"/healthz", "/readyz"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
			t.Fatalf("unexpected health response for %s: %d %s", path, recorder.Code, recorder.Body.String())
		}
	}

	modelsRecorder := httptest.NewRecorder()
	handler.ServeHTTP(modelsRecorder, httptest.NewRequest(http.MethodGet, "/v1/models", nil))
	if modelsRecorder.Code != http.StatusOK || !strings.Contains(modelsRecorder.Body.String(), `"id":"google/gemma-4-31B-it"`) {
		t.Fatalf("unexpected models response: %d %s", modelsRecorder.Code, modelsRecorder.Body.String())
	}

	modelRecorder := httptest.NewRecorder()
	handler.ServeHTTP(modelRecorder, httptest.NewRequest(http.MethodGet, "/v1/models/google/gemma-4-31B-it", nil))
	if modelRecorder.Code != http.StatusOK || !strings.Contains(modelRecorder.Body.String(), `"owned_by":"underpassai"`) {
		t.Fatalf("unexpected model response: %d %s", modelRecorder.Code, modelRecorder.Body.String())
	}

	missingRecorder := httptest.NewRecorder()
	handler.ServeHTTP(missingRecorder, httptest.NewRequest(http.MethodGet, "/v1/models/wrong", nil))
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestMappingHelpersCoverTwoPassAndErrors(t *testing.T) {
	response := domain.StructuredResponse{
		RequestID: "req-1",
		Output:    json.RawMessage(`{"ok":true}`),
		Metadata: domain.ResponseMetadata{
			ExecutionMode:      domain.ExecutionModeTwoPass,
			SchemaVersion:      "v1",
			Pass1PromptVersion: "p1",
			Pass2PromptVersion: "p2",
			IRVersion:          "1.0.0",
			Pass1:              domain.PassMetrics{Model: "m1"},
			Pass2:              domain.PassMetrics{Model: "m2", FinishReason: "stop", PromptTokens: 3, CompletionTokens: 4},
		},
	}

	dto := mapStructuredResponseDomainToDTO(response)
	if dto.Metadata.Pass1 == nil || dto.Metadata.Pass2 == nil || dto.Metadata.SinglePass != nil {
		t.Fatalf("unexpected metadata mapping: %#v", dto.Metadata)
	}
	if mapped := mapDomainErrorToDTO(&domain.Error{Code: domain.ErrorCodeInvalidRequest, Message: "bad"}); mapped.Error.Code != "invalid_request" {
		t.Fatalf("unexpected error mapping: %#v", mapped)
	}
	if got := mapModelNamePtr(nil); got != nil {
		t.Fatalf("expected nil model ptr")
	}
	if got := mapReasoningEffortPtr(nil); got != nil {
		t.Fatalf("expected nil reasoning ptr")
	}
}

func TestOpenAIMapperHelpers(t *testing.T) {
	content, err := flattenOpenAIMessageContent(json.RawMessage(`[{"type":"text","text":"hello "},{"type":"refusal","refusal":"world"}]`))
	if err != nil || content != "hello world" {
		t.Fatalf("unexpected flattened content: %q err=%v", content, err)
	}
	if _, err := flattenOpenAIMessageContent(json.RawMessage(`[{"type":"image"}]`)); err == nil {
		t.Fatalf("expected unsupported part type error")
	}
	if _, _, err := schemaFromOpenAIResponseFormat(&openAIResponseFormatDTO{Type: "json_object"}); err != nil {
		t.Fatalf("unexpected json_object error: %v", err)
	}
	if _, _, err := schemaFromOpenAIResponseFormat(&openAIResponseFormatDTO{Type: "bad"}); err == nil {
		t.Fatalf("expected unsupported response_format error")
	}
	if err := validateOpenAIModel("", "model"); err == nil {
		t.Fatalf("expected missing model error")
	}
	if err := validateOpenAIModel("model", ""); err != nil {
		t.Fatalf("unexpected validation error with empty public model: %v", err)
	}
	if _, err := resolveOpenAIMaxTokens(intPtr(1), intPtr(2)); err == nil {
		t.Fatalf("expected max token mismatch error")
	}
	if !isSupportedOpenAIRole("assistant") || isSupportedOpenAIRole("tool") {
		t.Fatalf("unexpected role support")
	}

	singleUsage := mapUsageFromMetadata(domain.ResponseMetadata{
		ExecutionMode: domain.ExecutionModeSinglePass,
		SinglePass:    domain.PassMetrics{PromptTokens: 2, CompletionTokens: 3, FinishReason: "stop", Model: "single"},
	})
	if singleUsage.TotalTokens != 5 {
		t.Fatalf("unexpected single-pass usage: %#v", singleUsage)
	}
	twoUsage := mapUsageFromMetadata(domain.ResponseMetadata{
		ExecutionMode: domain.ExecutionModeTwoPass,
		Pass1:         domain.PassMetrics{PromptTokens: 2, CompletionTokens: 3, Model: "p1"},
		Pass2:         domain.PassMetrics{PromptTokens: 4, CompletionTokens: 5, FinishReason: "stop", Model: "p2"},
	})
	if twoUsage.TotalTokens != 14 {
		t.Fatalf("unexpected two-pass usage: %#v", twoUsage)
	}
	if resolveResponseModel(domain.ResponseMetadata{ExecutionMode: domain.ExecutionModeSinglePass, SinglePass: domain.PassMetrics{Model: "single"}}, "fallback") != "single" {
		t.Fatalf("unexpected single-pass response model")
	}
	if resolveResponseModel(domain.ResponseMetadata{ExecutionMode: domain.ExecutionModeTwoPass, Pass2: domain.PassMetrics{Model: "p2"}}, "fallback") != "p2" {
		t.Fatalf("unexpected two-pass response model")
	}
	if resolveFinishReason(domain.ResponseMetadata{ExecutionMode: domain.ExecutionModeSinglePass, SinglePass: domain.PassMetrics{FinishReason: "stop"}}) != "stop" {
		t.Fatalf("unexpected finish reason")
	}
	if buildOpenAIModelInfoDTO("model").ID != "model" {
		t.Fatalf("unexpected model info dto")
	}
}

func TestResponsesMapperHelpers(t *testing.T) {
	input, err := flattenResponsesInput(json.RawMessage(`[{"role":"user","content":"hello"},{"role":"assistant","content":"world"}]`))
	if err != nil {
		t.Fatalf("unexpected flattenResponsesInput error: %v", err)
	}
	if !strings.Contains(input, "USER:\nhello") || !strings.Contains(input, "ASSISTANT:\nworld") {
		t.Fatalf("unexpected flattened input: %q", input)
	}
	if _, err := flattenResponsesInput(json.RawMessage(`[{"type":"image","content":"bad"}]`)); err == nil {
		t.Fatalf("expected invalid input item type error")
	}
	if _, _, err := schemaFromResponsesTextFormat(&openAIResponsesTextDTO{Format: &openAIResponsesFormatDTO{Type: "json_object"}}); err != nil {
		t.Fatalf("unexpected json_object error: %v", err)
	}
	if _, _, err := schemaFromResponsesTextFormat(&openAIResponsesTextDTO{Format: &openAIResponsesFormatDTO{Type: "text"}}); err == nil {
		t.Fatalf("expected unsupported text.format.type error")
	}
	if normalizeResponsesInstructions(json.RawMessage(`"hello"`)).(string) != "hello" {
		t.Fatalf("unexpected normalized instructions")
	}
	if normalizeResponsesFormat(nil).Type != "json_schema" {
		t.Fatalf("unexpected default format")
	}
	event := mapOpenAIErrorToResponsesEventDTO(3, openAIErrorEnvelopeDTO{Error: openAIErrorDTO{Message: "boom"}})
	if event.SequenceNumber != 3 || event.Error.Message != "boom" {
		t.Fatalf("unexpected error event: %#v", event)
	}
}

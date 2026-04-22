package httpapi

import (
	"encoding/json"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestMapStructuredRequestDTOToDomainMapsSinglePassOverrides(t *testing.T) {
	dto := structuredRequestDTO{
		Input:  "hello",
		Schema: json.RawMessage(`{"type":"object"}`),
		SinglePass: &passOverridesDTO{
			Model:     stringPtr("google/gemma-4-31B-it"),
			MaxTokens: intPtr(512),
		},
	}

	request := mapStructuredRequestDTOToDomain(dto)
	if request.SinglePass == nil {
		t.Fatalf("expected single_pass overrides to be mapped")
	}
	if request.SinglePass.Model == nil || *request.SinglePass.Model != domain.ModelName("google/gemma-4-31B-it") {
		t.Fatalf("unexpected single_pass model override: %#v", request.SinglePass.Model)
	}
	if request.SinglePass.MaxTokens == nil || *request.SinglePass.MaxTokens != 512 {
		t.Fatalf("unexpected single_pass max_tokens override: %#v", request.SinglePass.MaxTokens)
	}
}

func TestMapStructuredResponseDomainToDTOSinglePassUsesSinglePassFields(t *testing.T) {
	response := domain.StructuredResponse{
		RequestID: "req-1",
		Output:    json.RawMessage(`{"ok":true}`),
		Metadata: domain.ResponseMetadata{
			ExecutionMode:           domain.ExecutionModeSinglePass,
			SchemaVersion:           "v1",
			SinglePassPromptVersion: "sp1",
			IRVersion:               "1.0.0",
			SinglePass: domain.PassMetrics{
				Model:            "google/gemma-4-31B-it",
				Attempts:         1,
				LatencyMs:        1234,
				PromptTokens:     10,
				CompletionTokens: 20,
				FinishReason:     "stop",
				ContentPresent:   true,
			},
		},
	}

	dto := mapStructuredResponseDomainToDTO(response)
	if dto.Metadata.SinglePassPromptVersion != "sp1" {
		t.Fatalf("unexpected single_pass_prompt_version: %q", dto.Metadata.SinglePassPromptVersion)
	}
	if dto.Metadata.SinglePass == nil {
		t.Fatalf("expected single_pass metrics to be present")
	}
	if dto.Metadata.Pass1 != nil || dto.Metadata.Pass2 != nil {
		t.Fatalf("expected pass1/pass2 metrics to be omitted for single_pass responses")
	}
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

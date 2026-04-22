package twopassapp

import (
	"encoding/json"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestValidateRequestRejectsInvalidSchema(t *testing.T) {
	err := validateRequest(domain.StructuredRequest{
		Input:  "hello",
		Schema: json.RawMessage(`{"type":"object"`),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err.Code != domain.ErrorCodeInvalidRequest {
		t.Fatalf("unexpected error code: %q", err.Code)
	}
}

func TestValidateRequestAcceptsMinimalValidRequest(t *testing.T) {
	err := validateRequest(domain.StructuredRequest{
		Input:  "hello",
		Schema: json.RawMessage(`{"type":"object"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateIntermediateSizeRejectsOversizedIR(t *testing.T) {
	err := validateIntermediateSize(domain.IntermediateRepresentation("abcdef"), 5)
	if err == nil {
		t.Fatalf("expected size validation error")
	}
	if err.Code != domain.ErrorCodePass1TooLarge {
		t.Fatalf("unexpected error code: %q", err.Code)
	}
}

func TestValidateIntermediateSizeAcceptsBoundarySize(t *testing.T) {
	err := validateIntermediateSize(domain.IntermediateRepresentation("abcde"), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

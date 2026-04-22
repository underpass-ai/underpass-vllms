package jsonschemaadapter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	if NewValidator() == nil {
		t.Fatalf("expected validator instance")
	}
}

func TestValidateSuccess(t *testing.T) {
	validator := NewValidator()
	schema := json.RawMessage(`{
		"type":"object",
		"properties":{"value":{"type":"string"}},
		"required":["value"],
		"additionalProperties":false
	}`)
	candidate := json.RawMessage(`{"value":"ok"}`)

	if err := validator.Validate(schema, candidate); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFailsOnInvalidCandidateJSON(t *testing.T) {
	validator := NewValidator()
	schema := json.RawMessage(`{"type":"object"}`)
	candidate := json.RawMessage(`{"value":`)

	err := validator.Validate(schema, candidate)
	if err == nil {
		t.Fatalf("expected error for invalid candidate JSON")
	}
	if !strings.Contains(err.Error(), "decode candidate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFailsOnSchemaViolation(t *testing.T) {
	validator := NewValidator()
	schema := json.RawMessage(`{
		"type":"object",
		"properties":{"value":{"type":"string"}},
		"required":["value"],
		"additionalProperties":false
	}`)
	candidate := json.RawMessage(`{"value":123}`)

	err := validator.Validate(schema, candidate)
	if err == nil {
		t.Fatalf("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

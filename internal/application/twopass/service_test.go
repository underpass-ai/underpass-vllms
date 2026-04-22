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

const (
	testPass1SystemPrompt  = "You are a semantic extraction engine."
	testPass2SystemPrompt  = "You are a strict JSON canonicalization engine."
	testPass1UserTemplate  = "Extract facts from the following input and return only the intermediate representation.\n\nInput:\n{{input}}"
	testPass2UserTemplate  = "Convert the following intermediate representation into the target schema.\n\nTarget JSON schema:\n{{schema}}\n\nIntermediate representation:\n{{intermediate}}"
	testPass2RetryTemplate = "Previous attempt failed validation. Correct the output using this feedback:\n{{hint}}"
)

type completionStub struct {
	responses []domain.CompletionResponse
	errors    []error
	calls     int
}

func (s *completionStub) Complete(_ context.Context, _ domain.CompletionRequest) (domain.CompletionResponse, error) {
	index := s.calls
	s.calls++
	if index < len(s.errors) && s.errors[index] != nil {
		return domain.CompletionResponse{}, s.errors[index]
	}
	if index < len(s.responses) {
		return s.responses[index], nil
	}
	return domain.CompletionResponse{}, errors.New("unexpected call")
}

type validatorStub struct {
	errors []error
	calls  int
}

func (s *validatorStub) Validate(_ json.RawMessage, _ json.RawMessage) error {
	index := s.calls
	s.calls++
	if index < len(s.errors) {
		return s.errors[index]
	}
	return nil
}

func TestExecuteSuccess(t *testing.T) {
	reasoner := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: "Invoice number: INV-1\nWarnings:\n- none",
				Usage:   domain.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			},
		},
	}
	formatter := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: `{"invoice_number":"INV-1","warnings":["none"]}`,
				Usage:   domain.Usage{PromptTokens: 6, CompletionTokens: 4, TotalTokens: 10},
			},
		},
	}
	validator := &validatorStub{}
	logger := log.New(io.Discard, "", 0)
	adapter := NewTwoPassAdapter(
		Settings{
			MaxIntermediateBytes: 1024,
			Pass2RetryCount:      1,
			Versions: Versions{
				Pass1Prompt: "p1",
				Pass2Prompt: "p2",
				IR:          "1.0.0",
			},
			PromptTemplates: PromptTemplates{Pass1User: testPass1UserTemplate, Pass2User: testPass2UserTemplate, Pass2RetryHint: testPass2RetryTemplate},
			Pass1:           PassDefaults{Model: "reasoner", SystemPrompt: testPass1SystemPrompt, Temperature: 0.2, MaxTokens: 100},
			Pass2:           PassDefaults{Model: "formatter", SystemPrompt: testPass2SystemPrompt, Temperature: 0, MaxTokens: 100},
		},
		reasoner,
		formatter,
		validator,
		logger,
	)

	service := NewService(adapter)

	response, execErr := service.Execute(context.Background(), domain.StructuredRequest{
		Input:         "Invoice INV-1",
		SchemaVersion: "invoice-v1",
		Schema:        json.RawMessage(`{"type":"object"}`),
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if response.Metadata.Pass2.Attempts != 1 {
		t.Fatalf("expected a single pass2 attempt, got %d", response.Metadata.Pass2.Attempts)
	}
	if string(response.Output) != `{"invoice_number":"INV-1","warnings":["none"]}` {
		t.Fatalf("unexpected output: %s", string(response.Output))
	}
}

func TestExecuteRetriesPass2OnValidationFailure(t *testing.T) {
	reasoner := &completionStub{
		responses: []domain.CompletionResponse{
			{Content: "Invoice number: INV-1", Usage: domain.Usage{}},
		},
	}
	formatter := &completionStub{
		responses: []domain.CompletionResponse{
			{Content: `{"invoice_number":1}`},
			{Content: `{"invoice_number":"INV-1"}`},
		},
	}
	validator := &validatorStub{
		errors: []error{
			errors.New("schema validation failed"),
			nil,
		},
	}
	logger := log.New(io.Discard, "", 0)
	adapter := NewTwoPassAdapter(
		Settings{
			MaxIntermediateBytes: 1024,
			Pass2RetryCount:      1,
			Versions: Versions{
				Pass1Prompt: "p1",
				Pass2Prompt: "p2",
				IR:          "1.0.0",
			},
			PromptTemplates: PromptTemplates{Pass1User: testPass1UserTemplate, Pass2User: testPass2UserTemplate, Pass2RetryHint: testPass2RetryTemplate},
			Pass1:           PassDefaults{Model: "reasoner", SystemPrompt: testPass1SystemPrompt, Temperature: 0.2, MaxTokens: 100},
			Pass2:           PassDefaults{Model: "formatter", SystemPrompt: testPass2SystemPrompt, Temperature: 0, MaxTokens: 100},
		},
		reasoner,
		formatter,
		validator,
		logger,
	)

	service := NewService(adapter)

	response, execErr := service.Execute(context.Background(), domain.StructuredRequest{
		Input:  "Invoice INV-1",
		Schema: json.RawMessage(`{"type":"object"}`),
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if response.Metadata.Pass2.Attempts != 2 {
		t.Fatalf("expected two pass2 attempts, got %d", response.Metadata.Pass2.Attempts)
	}
}

func TestExecuteFallsBackToReasoningWhenPass1ContentIsEmpty(t *testing.T) {
	reasoner := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content:      "",
				Reasoning:    "Bug facts from reasoning trace",
				FinishReason: "length",
				Usage:        domain.Usage{PromptTokens: 10, CompletionTokens: 100, TotalTokens: 110},
			},
		},
	}
	formatter := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: `{"kind":"bug","warnings":[]}`,
				Usage:   domain.Usage{PromptTokens: 6, CompletionTokens: 4, TotalTokens: 10},
			},
		},
	}
	validator := &validatorStub{}
	logger := log.New(io.Discard, "", 0)
	adapter := NewTwoPassAdapter(
		Settings{
			MaxIntermediateBytes: 1024,
			Pass2RetryCount:      1,
			Versions: Versions{
				Pass1Prompt: "p1",
				Pass2Prompt: "p2",
				IR:          "1.0.0",
			},
			PromptTemplates: PromptTemplates{Pass1User: testPass1UserTemplate, Pass2User: testPass2UserTemplate, Pass2RetryHint: testPass2RetryTemplate},
			Pass1:           PassDefaults{Model: "reasoner", SystemPrompt: testPass1SystemPrompt, Temperature: 0.2, MaxTokens: 100},
			Pass2:           PassDefaults{Model: "formatter", SystemPrompt: testPass2SystemPrompt, Temperature: 0, MaxTokens: 100},
		},
		reasoner,
		formatter,
		validator,
		logger,
	)

	service := NewService(adapter)

	response, execErr := service.Execute(context.Background(), domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if response.IntermediateRepresentation != "Bug facts from reasoning trace" {
		t.Fatalf("expected intermediate representation from reasoning fallback, got %q", response.IntermediateRepresentation)
	}
	if !response.Metadata.Pass1.UsedReasoningFallback {
		t.Fatalf("expected pass1 to report reasoning fallback")
	}
	if !response.Metadata.Pass1.ReasoningPresent {
		t.Fatalf("expected pass1 reasoning to be marked as present")
	}
	if response.Metadata.Pass1.ContentPresent {
		t.Fatalf("expected pass1 content to be marked as absent")
	}
	if !response.Metadata.Pass1.Truncated {
		t.Fatalf("expected pass1 to be marked as truncated")
	}
}

func TestExecuteSanitizesMarkdownWrappedIntermediate(t *testing.T) {
	reasoner := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: "**Reasoning:** brief note\n\n**Final Intermediate Representation:**\n```json\n{\"kind\":\"bug\",\"severity\":\"High\"}\n```",
				Usage:   domain.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
			},
		},
	}
	formatter := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: `{"kind":"bug","severity":"High","warnings":[]}`,
				Usage:   domain.Usage{PromptTokens: 6, CompletionTokens: 4, TotalTokens: 10},
			},
		},
	}
	validator := &validatorStub{}
	logger := log.New(io.Discard, "", 0)
	adapter := NewTwoPassAdapter(
		Settings{
			MaxIntermediateBytes: 1024,
			Pass2RetryCount:      1,
			Versions: Versions{
				Pass1Prompt: "p1",
				Pass2Prompt: "p2",
				IR:          "1.0.0",
			},
			PromptTemplates: PromptTemplates{Pass1User: testPass1UserTemplate, Pass2User: testPass2UserTemplate, Pass2RetryHint: testPass2RetryTemplate},
			Pass1:           PassDefaults{Model: "reasoner", SystemPrompt: testPass1SystemPrompt, Temperature: 0.2, MaxTokens: 100},
			Pass2:           PassDefaults{Model: "formatter", SystemPrompt: testPass2SystemPrompt, Temperature: 0, MaxTokens: 100},
		},
		reasoner,
		formatter,
		validator,
		logger,
	)

	service := NewService(adapter)

	response, execErr := service.Execute(context.Background(), domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if response.IntermediateRepresentation != "{\"kind\":\"bug\",\"severity\":\"High\"}" {
		t.Fatalf("expected sanitized intermediate representation, got %q", response.IntermediateRepresentation)
	}
}

func TestExecuteSuccessWithSinglePassAdapter(t *testing.T) {
	formatter := &completionStub{
		responses: []domain.CompletionResponse{
			{
				Content: `{"kind":"bug","warnings":[]}`,
				Usage:   domain.Usage{PromptTokens: 8, CompletionTokens: 5, TotalTokens: 13},
			},
		},
	}
	logger := log.New(io.Discard, "", 0)
	adapter := NewSinglePassAdapter(
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
			SinglePass: PassDefaults{Model: "gpt-oss-20b", Temperature: 0.2, MaxTokens: 256},
		},
		formatter,
		logger,
	)
	service := NewService(adapter)

	response, execErr := service.Execute(context.Background(), domain.StructuredRequest{
		Input:  "Bug report",
		Schema: json.RawMessage(`{"type":"object"}`),
	})
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}
	if formatter.calls != 1 {
		t.Fatalf("expected single formatter call, got %d", formatter.calls)
	}
	if response.Metadata.ExecutionMode != domain.ExecutionModeSinglePass {
		t.Fatalf("unexpected execution mode: %q", response.Metadata.ExecutionMode)
	}
	if response.Metadata.SinglePassPromptVersion != "sp1" {
		t.Fatalf("unexpected single-pass prompt version: %q", response.Metadata.SinglePassPromptVersion)
	}
	if response.Metadata.SinglePass.Attempts != 1 {
		t.Fatalf("expected a single single-pass attempt, got %d", response.Metadata.SinglePass.Attempts)
	}
	if response.IntermediateRepresentation != "" {
		t.Fatalf("expected empty intermediate representation for single-pass, got %q", response.IntermediateRepresentation)
	}
}

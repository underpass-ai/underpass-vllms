package twopassapp

import (
	"strings"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestBuildPass2PromptEmphasizesSpecificAttributes(t *testing.T) {
	template := "Convert the following intermediate representation into the target schema.\n\nRules:\n- Prefer specific extracted attributes over generic entity labels or types.\n- If the intermediate representation contains both an entity type and a more specific field value, use the more specific field value.\n- Output exactly one JSON value that matches the target schema.\n\nTarget JSON schema:\n{{schema}}\n\nIntermediate representation:\n{{intermediate}}"
	prompt := buildPass2Prompt(template, "Previous attempt failed validation. Correct the output using this feedback:\n{{hint}}", domain.IntermediateRepresentation("Entity: Ada Lovelace"), []byte(`{"type":"object","properties":{"name":{"type":"string"}}}`), "")

	requiredSnippets := []string{
		"Prefer specific extracted attributes over generic entity labels or types.",
		"use the more specific field value",
		"Output exactly one JSON value that matches the target schema.",
		`"properties":{"name":{"type":"string"}}`,
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildPass1PromptRequestsCompactFinalIR(t *testing.T) {
	template := "Given the following input, extract all facts required for the downstream schema.\n\nRequirements:\n- Keep any reasoning brief and keep it out of the final answer.\n- The final answer must contain only the intermediate representation.\n- End with a compact final intermediate representation that can be passed to a JSON canonicalizer.\n\nInput:\n{{input}}"
	prompt := buildPass1Prompt(template, domain.InputPayload("bug report"))

	requiredSnippets := []string{
		"Keep any reasoning brief and keep it out of the final answer.",
		"The final answer must contain only the intermediate representation.",
		"End with a compact final intermediate representation that can be passed to a JSON canonicalizer.",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
}

func TestBuildPass2PromptAppendsRetryHintTemplate(t *testing.T) {
	template := "Schema:\n{{schema}}\nIR:\n{{intermediate}}"
	retryTemplate := "Retry with this hint:\n{{hint}}"

	prompt := buildPass2Prompt(template, retryTemplate, domain.IntermediateRepresentation("ir"), []byte(`{"type":"object"}`), "field mismatch")

	if !strings.Contains(prompt, "Retry with this hint:\nfield mismatch") {
		t.Fatalf("expected retry hint template to be rendered, got %q", prompt)
	}
}

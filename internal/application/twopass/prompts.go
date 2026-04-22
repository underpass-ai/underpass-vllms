package twopassapp

import (
	"encoding/json"
	"strings"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func buildPass1Prompt(template string, input domain.InputPayload) string {
	return renderPromptTemplate(template, map[string]string{
		"input": string(input),
	})
}

func buildPass2Prompt(template, retryHintTemplate string, intermediate domain.IntermediateRepresentation, schema json.RawMessage, hint string) string {
	base := renderPromptTemplate(template, map[string]string{
		"schema":       string(schema),
		"intermediate": string(intermediate),
	})
	if strings.TrimSpace(hint) == "" {
		return base
	}
	return base + "\n\n" + renderPromptTemplate(retryHintTemplate, map[string]string{
		"hint": hint,
	})
}

func buildSinglePassPrompt(template, retryHintTemplate string, input domain.InputPayload, schema json.RawMessage, hint string) string {
	base := renderPromptTemplate(template, map[string]string{
		"input":  string(input),
		"schema": string(schema),
	})
	if strings.TrimSpace(hint) == "" {
		return base
	}
	return base + "\n\n" + renderPromptTemplate(retryHintTemplate, map[string]string{
		"hint": hint,
	})
}

func renderPromptTemplate(template string, values map[string]string) string {
	rendered := template
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return rendered
}

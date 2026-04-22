package main

import (
	"testing"
	"time"

	"github.com/tgarciai/underpass-vllms/internal/config"
	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestMapEndpointConfigToPassDefaults(t *testing.T) {
	topP := 0.9
	topK := 20
	presence := 0.1
	repetition := 1.1
	thinkingBudget := 2048
	reasoningEffort := "high"
	preserveThinking := true

	defaults := mapEndpointConfigToPassDefaults(config.EndpointConfig{
		Model:               "google/gemma-4-31B-it",
		SystemPrompt:        "system",
		Temperature:         0.2,
		TopP:                &topP,
		TopK:                &topK,
		PresencePenalty:     &presence,
		RepetitionPenalty:   &repetition,
		MaxTokens:           4096,
		ThinkingTokenBudget: &thinkingBudget,
		ReasoningEffort:     &reasoningEffort,
		PreserveThinking:    &preserveThinking,
		Timeout:             45 * time.Second,
	})

	if defaults.Model != domain.ModelName("google/gemma-4-31B-it") {
		t.Fatalf("unexpected model: %q", defaults.Model)
	}
	if defaults.SystemPrompt != "system" || defaults.Temperature != 0.2 || defaults.MaxTokens != 4096 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}
	if defaults.TopP == nil || *defaults.TopP != topP {
		t.Fatalf("unexpected top_p: %#v", defaults.TopP)
	}
	if defaults.TopK == nil || *defaults.TopK != topK {
		t.Fatalf("unexpected top_k: %#v", defaults.TopK)
	}
	if defaults.ReasoningEffort == nil || *defaults.ReasoningEffort != domain.ReasoningEffort("high") {
		t.Fatalf("unexpected reasoning effort: %#v", defaults.ReasoningEffort)
	}
	if defaults.PreserveThinking == nil || *defaults.PreserveThinking != true {
		t.Fatalf("unexpected preserve thinking: %#v", defaults.PreserveThinking)
	}
}

func TestMapReasoningEffort(t *testing.T) {
	if mapReasoningEffort(nil) != nil {
		t.Fatalf("expected nil mapping for nil input")
	}
	value := "medium"
	mapped := mapReasoningEffort(&value)
	if mapped == nil || *mapped != domain.ReasoningEffort("medium") {
		t.Fatalf("unexpected mapping: %#v", mapped)
	}
}

func TestResolvePublicModel(t *testing.T) {
	if got := resolvePublicModel(config.Settings{
		Pass2: config.EndpointConfig{Model: "pass2-model"},
		Pass1: config.EndpointConfig{Model: "pass1-model"},
	}); got != "pass2-model" {
		t.Fatalf("unexpected public model: %q", got)
	}
	if got := resolvePublicModel(config.Settings{
		Pass1: config.EndpointConfig{Model: "pass1-model"},
	}); got != "pass1-model" {
		t.Fatalf("unexpected public model fallback: %q", got)
	}
	if got := resolvePublicModel(config.Settings{}); got != "underpass-orchestrator" {
		t.Fatalf("unexpected default public model: %q", got)
	}
}

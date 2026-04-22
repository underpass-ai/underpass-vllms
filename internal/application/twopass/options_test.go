package twopassapp

import (
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func TestResolvePassDefaultsAppliesAllOverrides(t *testing.T) {
	topP := 0.8
	topK := 20
	presence := 0.1
	repetition := 1.1
	maxTokens := 512
	thinkingBudget := 128
	reasoningEffort := domain.ReasoningEffort("high")
	preserveThinking := true
	model := domain.ModelName("override-model")
	systemPrompt := "override-system"

	resolved := resolvePassDefaults(
		PassDefaults{
			Model:               "default-model",
			SystemPrompt:        "default-system",
			Temperature:         0.2,
			MaxTokens:           256,
			ThinkingTokenBudget: nil,
		},
		&domain.PassOverrides{
			Model:               &model,
			SystemPrompt:        &systemPrompt,
			Temperature:         floatPtrTP(0.6),
			TopP:                &topP,
			TopK:                &topK,
			PresencePenalty:     &presence,
			RepetitionPenalty:   &repetition,
			MaxTokens:           &maxTokens,
			ThinkingTokenBudget: &thinkingBudget,
			ReasoningEffort:     &reasoningEffort,
			PreserveThinking:    &preserveThinking,
		},
	)

	if resolved.Model != "override-model" || resolved.SystemPrompt != "override-system" {
		t.Fatalf("unexpected resolved defaults: %+v", resolved)
	}
	if resolved.Temperature != 0.6 || resolved.MaxTokens != 512 {
		t.Fatalf("unexpected resolved defaults: %+v", resolved)
	}
	if resolved.TopP == nil || *resolved.TopP != topP {
		t.Fatalf("unexpected top_p: %#v", resolved.TopP)
	}
	if resolved.TopK == nil || *resolved.TopK != topK {
		t.Fatalf("unexpected top_k: %#v", resolved.TopK)
	}
	if resolved.PresencePenalty == nil || *resolved.PresencePenalty != presence {
		t.Fatalf("unexpected presence penalty: %#v", resolved.PresencePenalty)
	}
	if resolved.RepetitionPenalty == nil || *resolved.RepetitionPenalty != repetition {
		t.Fatalf("unexpected repetition penalty: %#v", resolved.RepetitionPenalty)
	}
	if resolved.ThinkingTokenBudget == nil || *resolved.ThinkingTokenBudget != thinkingBudget {
		t.Fatalf("unexpected thinking budget: %#v", resolved.ThinkingTokenBudget)
	}
	if resolved.ReasoningEffort == nil || *resolved.ReasoningEffort != reasoningEffort {
		t.Fatalf("unexpected reasoning effort: %#v", resolved.ReasoningEffort)
	}
	if resolved.PreserveThinking == nil || *resolved.PreserveThinking != true {
		t.Fatalf("unexpected preserve thinking: %#v", resolved.PreserveThinking)
	}
}

func TestResolveSinglePassOptionsPrefersSinglePassOverrides(t *testing.T) {
	singlePassModel := domain.ModelName("single-pass-model")
	legacyModel := domain.ModelName("legacy-model")

	resolved := resolveSinglePassOptions(
		PassDefaults{Model: "default-model", SystemPrompt: "default-system"},
		"single-pass-system",
		&domain.PassOverrides{Model: &singlePassModel},
		&domain.PassOverrides{Model: &legacyModel},
	)

	if resolved.Model != "single-pass-model" {
		t.Fatalf("unexpected resolved model: %q", resolved.Model)
	}
	if resolved.SystemPrompt != "single-pass-system" {
		t.Fatalf("unexpected system prompt: %q", resolved.SystemPrompt)
	}
}

func TestResolvePassDefaultsIgnoresBlankSystemPromptOverride(t *testing.T) {
	blank := "   "

	resolved := resolvePassDefaults(
		PassDefaults{
			Model:        "default-model",
			SystemPrompt: "default-system",
			Temperature:  0.2,
			MaxTokens:    256,
		},
		&domain.PassOverrides{
			SystemPrompt: &blank,
		},
	)

	if resolved.SystemPrompt != "default-system" {
		t.Fatalf("unexpected system prompt: %q", resolved.SystemPrompt)
	}
}

func TestResolvePassDefaultsReturnsDefaultsWhenOverridesAreNil(t *testing.T) {
	defaults := PassDefaults{
		Model:        "default-model",
		SystemPrompt: "default-system",
		Temperature:  0.2,
		MaxTokens:    256,
	}

	resolved := resolvePassDefaults(defaults, nil)

	if resolved != defaults {
		t.Fatalf("unexpected resolved defaults: %+v", resolved)
	}
}

func floatPtrTP(value float64) *float64 {
	return &value
}

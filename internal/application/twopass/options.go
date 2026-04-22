package twopassapp

import (
	"strings"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func resolveSinglePassOptions(defaults PassDefaults, systemPrompt string, overrides *domain.PassOverrides, legacyOverrides *domain.PassOverrides) PassDefaults {
	resolved := defaults
	resolved.SystemPrompt = systemPrompt
	if overrides != nil {
		return resolvePassDefaults(resolved, overrides)
	}
	return resolvePassDefaults(resolved, legacyOverrides)
}

func resolvePassDefaults(defaults PassDefaults, overrides *domain.PassOverrides) PassDefaults {
	resolved := defaults
	if overrides == nil {
		return resolved
	}
	if overrides.Model != nil {
		resolved.Model = *overrides.Model
	}
	if overrides.SystemPrompt != nil && strings.TrimSpace(*overrides.SystemPrompt) != "" {
		resolved.SystemPrompt = *overrides.SystemPrompt
	}
	if overrides.Temperature != nil {
		resolved.Temperature = *overrides.Temperature
	}
	if overrides.TopP != nil {
		resolved.TopP = overrides.TopP
	}
	if overrides.TopK != nil {
		resolved.TopK = overrides.TopK
	}
	if overrides.PresencePenalty != nil {
		resolved.PresencePenalty = overrides.PresencePenalty
	}
	if overrides.RepetitionPenalty != nil {
		resolved.RepetitionPenalty = overrides.RepetitionPenalty
	}
	if overrides.MaxTokens != nil {
		resolved.MaxTokens = *overrides.MaxTokens
	}
	if overrides.ThinkingTokenBudget != nil {
		resolved.ThinkingTokenBudget = overrides.ThinkingTokenBudget
	}
	if overrides.ReasoningEffort != nil {
		resolved.ReasoningEffort = overrides.ReasoningEffort
	}
	if overrides.PreserveThinking != nil {
		resolved.PreserveThinking = overrides.PreserveThinking
	}
	return resolved
}

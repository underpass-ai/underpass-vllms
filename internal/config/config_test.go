package config

import "testing"

func TestLoadFromEnvRequiresExternalPromptAndRuntimeConfig(t *testing.T) {
	setValidQwenEnv(t)
	t.Setenv("PASS1_SYSTEM_PROMPT", "")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatalf("expected error when PASS1_SYSTEM_PROMPT is missing")
	}
	if err.Error() != "PASS1_SYSTEM_PROMPT is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadFromEnvReadsQwenTwoPassConfig(t *testing.T) {
	setValidQwenEnv(t)

	settings, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if settings.ModelType != ModelTypeQwenReasoning {
		t.Fatalf("unexpected model type: %q", settings.ModelType)
	}
	if settings.Pass1.Provider != "vllm_chat_completions" {
		t.Fatalf("unexpected Pass1 provider: %q", settings.Pass1.Provider)
	}
	if settings.Pass2.Provider != "vllm_chat_completions" {
		t.Fatalf("unexpected Pass2 provider: %q", settings.Pass2.Provider)
	}
	if settings.Pass1.SystemPrompt != "pass1 system" {
		t.Fatalf("unexpected Pass1 system prompt: %q", settings.Pass1.SystemPrompt)
	}
	if settings.Pass2.SystemPrompt != "pass2 system" {
		t.Fatalf("unexpected Pass2 system prompt: %q", settings.Pass2.SystemPrompt)
	}
	if settings.Pass1UserPromptTemplate != "input={{input}}" {
		t.Fatalf("unexpected Pass1 template: %q", settings.Pass1UserPromptTemplate)
	}
	if settings.Pass2UserPromptTemplate != "schema={{schema}} ir={{intermediate}}" {
		t.Fatalf("unexpected Pass2 template: %q", settings.Pass2UserPromptTemplate)
	}
	if settings.Pass2RetryHintTemplate != "hint={{hint}}" {
		t.Fatalf("unexpected Pass2 retry template: %q", settings.Pass2RetryHintTemplate)
	}
	if settings.Pass1.Temperature != 0.6 {
		t.Fatalf("unexpected Pass1 temperature: %v", settings.Pass1.Temperature)
	}
	if settings.Pass1.TopP == nil || *settings.Pass1.TopP != 0.95 {
		t.Fatalf("unexpected Pass1 top_p: %#v", settings.Pass1.TopP)
	}
	if settings.Pass1.TopK == nil || *settings.Pass1.TopK != 20 {
		t.Fatalf("unexpected Pass1 top_k: %#v", settings.Pass1.TopK)
	}
	if settings.Pass1.ThinkingTokenBudget == nil || *settings.Pass1.ThinkingTokenBudget != 2048 {
		t.Fatalf("unexpected Pass1 thinking budget: %#v", settings.Pass1.ThinkingTokenBudget)
	}
	if settings.Pass1.PreserveThinking == nil || *settings.Pass1.PreserveThinking != true {
		t.Fatalf("unexpected Pass1 preserve_thinking: %#v", settings.Pass1.PreserveThinking)
	}
	if settings.Pass2.Temperature != 0 {
		t.Fatalf("unexpected Pass2 temperature: %v", settings.Pass2.Temperature)
	}
	if settings.Pass2.TopP != nil {
		t.Fatalf("expected Pass2 top_p to be nil, got %#v", settings.Pass2.TopP)
	}
}

func TestLoadFromEnvReadsGPTOssSinglePassConfig(t *testing.T) {
	setValidGPTOssEnv(t)

	settings, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if settings.ModelType != ModelTypeGPTOss {
		t.Fatalf("unexpected model type: %q", settings.ModelType)
	}
	if settings.Pass1.Provider != "" {
		t.Fatalf("expected empty Pass1 config for gpt-oss, got %+v", settings.Pass1)
	}
	if settings.Pass2.Provider != "openai_chat_completions" {
		t.Fatalf("unexpected Pass2 provider: %q", settings.Pass2.Provider)
	}
	if settings.SinglePassSystemPrompt != "single-pass system" {
		t.Fatalf("unexpected single-pass system prompt: %q", settings.SinglePassSystemPrompt)
	}
	if settings.SinglePassUserPromptTemplate != "schema={{schema}} input={{input}}" {
		t.Fatalf("unexpected single-pass user template: %q", settings.SinglePassUserPromptTemplate)
	}
	if settings.Pass2.ReasoningEffort == nil || *settings.Pass2.ReasoningEffort != "high" {
		t.Fatalf("unexpected Pass2 reasoning effort: %#v", settings.Pass2.ReasoningEffort)
	}
}

func TestLoadFromEnvReadsGemma4SinglePassConfig(t *testing.T) {
	setValidGemma4Env(t)

	settings, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if settings.ModelType != ModelTypeGemma4 {
		t.Fatalf("unexpected model type: %q", settings.ModelType)
	}
	if settings.Pass2.Provider != "vllm_chat_completions" {
		t.Fatalf("unexpected Pass2 provider: %q", settings.Pass2.Provider)
	}
	if settings.SinglePassSystemPrompt != "gemma system" {
		t.Fatalf("unexpected single-pass system prompt: %q", settings.SinglePassSystemPrompt)
	}
	if settings.SinglePassUserPromptTemplate != "schema={{schema}} input={{input}}" {
		t.Fatalf("unexpected single-pass user template: %q", settings.SinglePassUserPromptTemplate)
	}
}

func setValidQwenEnv(t *testing.T) {
	t.Helper()
	setCommonEnv(t)

	t.Setenv("MODEL_TYPE", ModelTypeQwenReasoning)
	t.Setenv("PASS1_PROMPT_VERSION", "2026-04-21.2")
	t.Setenv("PASS2_PROMPT_VERSION", "2026-04-21.1")

	t.Setenv("PASS1_PROVIDER", "vllm_chat_completions")
	t.Setenv("PASS1_BASE_URL", "http://reasoning/v1")
	t.Setenv("PASS1_MODEL", "reasoner-model")
	t.Setenv("PASS1_API_KEY", "EMPTY")
	t.Setenv("PASS1_SYSTEM_PROMPT", "pass1 system")
	t.Setenv("PASS1_USER_PROMPT_TEMPLATE", "input={{input}}")
	t.Setenv("PASS1_TEMPERATURE", "0.6")
	t.Setenv("PASS1_TOP_P", "0.95")
	t.Setenv("PASS1_TOP_K", "20")
	t.Setenv("PASS1_PRESENCE_PENALTY", "0")
	t.Setenv("PASS1_REPETITION_PENALTY", "1.0")
	t.Setenv("PASS1_MAX_TOKENS", "4096")
	t.Setenv("PASS1_THINKING_TOKEN_BUDGET", "2048")
	t.Setenv("PASS1_PRESERVE_THINKING", "true")
	t.Setenv("PASS1_TIMEOUT", "45s")

	t.Setenv("PASS2_SYSTEM_PROMPT", "pass2 system")
	t.Setenv("PASS2_USER_PROMPT_TEMPLATE", "schema={{schema}} ir={{intermediate}}")
}

func setValidGPTOssEnv(t *testing.T) {
	t.Helper()
	setCommonEnv(t)

	t.Setenv("MODEL_TYPE", ModelTypeGPTOss)
	t.Setenv("SINGLE_PASS_PROMPT_VERSION", "2026-04-21.1")
	t.Setenv("SINGLE_PASS_SYSTEM_PROMPT", "single-pass system")
	t.Setenv("SINGLE_PASS_USER_PROMPT_TEMPLATE", "schema={{schema}} input={{input}}")
	t.Setenv("PASS2_PROVIDER", "openai_chat_completions")
	t.Setenv("PASS2_REASONING_EFFORT", "high")
}

func setValidGemma4Env(t *testing.T) {
	t.Helper()
	setCommonEnv(t)

	t.Setenv("MODEL_TYPE", ModelTypeGemma4)
	t.Setenv("SINGLE_PASS_PROMPT_VERSION", "2026-04-21.1")
	t.Setenv("SINGLE_PASS_SYSTEM_PROMPT", "gemma system")
	t.Setenv("SINGLE_PASS_USER_PROMPT_TEMPLATE", "schema={{schema}} input={{input}}")
	t.Setenv("PASS2_PROVIDER", "vllm_chat_completions")
}

func setCommonEnv(t *testing.T) {
	t.Helper()

	t.Setenv("SERVER_ADDR", ":8080")
	t.Setenv("MAX_INTERMEDIATE_BYTES", "65536")
	t.Setenv("PASS2_RETRY_COUNT", "1")
	t.Setenv("IR_VERSION", "1.0.0")
	t.Setenv("PASS2_RETRY_HINT_TEMPLATE", "hint={{hint}}")

	t.Setenv("PASS2_PROVIDER", "vllm_chat_completions")
	t.Setenv("PASS2_BASE_URL", "http://structured/v1")
	t.Setenv("PASS2_MODEL", "formatter-model")
	t.Setenv("PASS2_API_KEY", "EMPTY")
	t.Setenv("PASS2_TEMPERATURE", "0")
	t.Setenv("PASS2_MAX_TOKENS", "4096")
	t.Setenv("PASS2_TIMEOUT", "45s")
}

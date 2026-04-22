package config

import "testing"

func TestLoadPass1ConfigRejectsNegativeThinkingBudget(t *testing.T) {
	setValidQwenEnv(t)
	t.Setenv("PASS1_THINKING_TOKEN_BUDGET", "-1")

	_, err := loadPass1Config()
	if err == nil {
		t.Fatalf("expected error for negative thinking budget")
	}
	if err.Error() != "PASS1_THINKING_TOKEN_BUDGET must be >= 0" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPass1ConfigTrimsBaseURL(t *testing.T) {
	setValidQwenEnv(t)
	t.Setenv("PASS1_BASE_URL", "http://reasoning/v1///")

	cfg, err := loadPass1Config()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BaseURL != "http://reasoning/v1" {
		t.Fatalf("unexpected base url: %q", cfg.BaseURL)
	}
}

func TestLoadPass2ConfigRequiresSystemPromptForQwen(t *testing.T) {
	setValidQwenEnv(t)
	t.Setenv("PASS2_SYSTEM_PROMPT", "")

	_, err := loadPass2Config(ModelTypeQwenReasoning)
	if err == nil {
		t.Fatalf("expected missing system prompt error")
	}
	if err.Error() != "PASS2_SYSTEM_PROMPT is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadPass2ConfigRejectsNegativeThinkingBudget(t *testing.T) {
	setValidGemma4Env(t)
	t.Setenv("PASS2_THINKING_TOKEN_BUDGET", "-1")

	_, err := loadPass2Config(ModelTypeGemma4)
	if err == nil {
		t.Fatalf("expected error for negative thinking budget")
	}
	if err.Error() != "PASS2_THINKING_TOKEN_BUDGET must be >= 0" {
		t.Fatalf("unexpected error: %v", err)
	}
}

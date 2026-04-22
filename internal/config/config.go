package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ModelTypeQwenReasoning = "qwen_reasoning"
	ModelTypeGPTOss        = "gpt_oss"
	ModelTypeGemma4        = "gemma4"
)

type EndpointConfig struct {
	Provider            string
	BaseURL             string
	APIKey              string
	Model               string
	SystemPrompt        string
	Temperature         float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	RepetitionPenalty   *float64
	MaxTokens           int
	ThinkingTokenBudget *int
	ReasoningEffort     *string
	PreserveThinking    *bool
	Timeout             time.Duration
}

type Settings struct {
	Addr                         string
	ModelType                    string
	MaxIntermediateBytes         int
	Pass2RetryCount              int
	Pass1PromptVersion           string
	Pass2PromptVersion           string
	SinglePassPromptVersion      string
	IRVersion                    string
	Pass1UserPromptTemplate      string
	Pass2UserPromptTemplate      string
	Pass2RetryHintTemplate       string
	SinglePassSystemPrompt       string
	SinglePassUserPromptTemplate string
	Pass1                        EndpointConfig
	Pass2                        EndpointConfig
}

func LoadFromEnv() (Settings, error) {
	addr, err := lookupRequiredString("SERVER_ADDR")
	if err != nil {
		return Settings{}, err
	}
	modelType, err := lookupRequiredString("MODEL_TYPE")
	if err != nil {
		return Settings{}, err
	}
	if err := validateModelType(modelType); err != nil {
		return Settings{}, err
	}

	maxIntermediateBytes, err := lookupRequiredInt("MAX_INTERMEDIATE_BYTES")
	if err != nil {
		return Settings{}, err
	}
	if maxIntermediateBytes <= 0 {
		return Settings{}, fmt.Errorf("MAX_INTERMEDIATE_BYTES must be > 0")
	}
	pass2RetryCount, err := lookupRequiredInt("PASS2_RETRY_COUNT")
	if err != nil {
		return Settings{}, err
	}
	if pass2RetryCount < 0 {
		return Settings{}, fmt.Errorf("PASS2_RETRY_COUNT must be >= 0")
	}
	irVersion, err := lookupRequiredString("IR_VERSION")
	if err != nil {
		return Settings{}, err
	}
	pass2RetryHintTemplate, err := lookupRequiredString("PASS2_RETRY_HINT_TEMPLATE")
	if err != nil {
		return Settings{}, err
	}

	settings := Settings{
		Addr:                   addr,
		ModelType:              modelType,
		MaxIntermediateBytes:   maxIntermediateBytes,
		Pass2RetryCount:        pass2RetryCount,
		IRVersion:              irVersion,
		Pass2RetryHintTemplate: pass2RetryHintTemplate,
	}

	pass2Config, err := loadPass2Config(modelType)
	if err != nil {
		return Settings{}, err
	}
	settings.Pass2 = pass2Config

	switch modelType {
	case ModelTypeQwenReasoning:
		settings.Pass1PromptVersion, err = lookupRequiredString("PASS1_PROMPT_VERSION")
		if err != nil {
			return Settings{}, err
		}
		settings.Pass2PromptVersion, err = lookupRequiredString("PASS2_PROMPT_VERSION")
		if err != nil {
			return Settings{}, err
		}
		settings.Pass1UserPromptTemplate, err = lookupRequiredString("PASS1_USER_PROMPT_TEMPLATE")
		if err != nil {
			return Settings{}, err
		}
		settings.Pass2UserPromptTemplate, err = lookupRequiredString("PASS2_USER_PROMPT_TEMPLATE")
		if err != nil {
			return Settings{}, err
		}
		settings.Pass1, err = loadPass1Config()
		if err != nil {
			return Settings{}, err
		}
	case ModelTypeGPTOss, ModelTypeGemma4:
		settings.SinglePassPromptVersion, err = lookupRequiredString("SINGLE_PASS_PROMPT_VERSION")
		if err != nil {
			return Settings{}, err
		}
		settings.SinglePassSystemPrompt, err = lookupRequiredString("SINGLE_PASS_SYSTEM_PROMPT")
		if err != nil {
			return Settings{}, err
		}
		settings.SinglePassUserPromptTemplate, err = lookupRequiredString("SINGLE_PASS_USER_PROMPT_TEMPLATE")
		if err != nil {
			return Settings{}, err
		}
	}

	return settings, nil
}

func loadPass1Config() (EndpointConfig, error) {
	provider, err := lookupRequiredString("PASS1_PROVIDER")
	if err != nil {
		return EndpointConfig{}, err
	}
	if err := validateProvider("PASS1_PROVIDER", provider); err != nil {
		return EndpointConfig{}, err
	}
	baseURL, err := lookupRequiredString("PASS1_BASE_URL")
	if err != nil {
		return EndpointConfig{}, err
	}
	apiKey, err := lookupRequiredString("PASS1_API_KEY")
	if err != nil {
		return EndpointConfig{}, err
	}
	model, err := lookupRequiredString("PASS1_MODEL")
	if err != nil {
		return EndpointConfig{}, err
	}
	systemPrompt, err := lookupRequiredString("PASS1_SYSTEM_PROMPT")
	if err != nil {
		return EndpointConfig{}, err
	}
	temperature, err := lookupRequiredFloat("PASS1_TEMPERATURE")
	if err != nil {
		return EndpointConfig{}, err
	}
	topP, err := lookupRequiredOptionalFloat("PASS1_TOP_P")
	if err != nil {
		return EndpointConfig{}, err
	}
	topK, err := lookupRequiredOptionalInt("PASS1_TOP_K")
	if err != nil {
		return EndpointConfig{}, err
	}
	presencePenalty, err := lookupRequiredOptionalFloat("PASS1_PRESENCE_PENALTY")
	if err != nil {
		return EndpointConfig{}, err
	}
	repetitionPenalty, err := lookupRequiredOptionalFloat("PASS1_REPETITION_PENALTY")
	if err != nil {
		return EndpointConfig{}, err
	}
	maxTokens, err := lookupRequiredInt("PASS1_MAX_TOKENS")
	if err != nil {
		return EndpointConfig{}, err
	}
	if maxTokens <= 0 {
		return EndpointConfig{}, fmt.Errorf("PASS1_MAX_TOKENS must be > 0")
	}
	thinkingTokenBudget, err := lookupRequiredOptionalInt("PASS1_THINKING_TOKEN_BUDGET")
	if err != nil {
		return EndpointConfig{}, err
	}
	if thinkingTokenBudget != nil && *thinkingTokenBudget < 0 {
		return EndpointConfig{}, fmt.Errorf("PASS1_THINKING_TOKEN_BUDGET must be >= 0")
	}
	reasoningEffort, err := lookupOptionalString("PASS1_REASONING_EFFORT")
	if err != nil {
		return EndpointConfig{}, err
	}
	preserveThinking, err := lookupRequiredOptionalBool("PASS1_PRESERVE_THINKING")
	if err != nil {
		return EndpointConfig{}, err
	}
	timeout, err := lookupRequiredDuration("PASS1_TIMEOUT")
	if err != nil {
		return EndpointConfig{}, err
	}
	if timeout <= 0 {
		return EndpointConfig{}, fmt.Errorf("PASS1_TIMEOUT must be > 0")
	}

	return EndpointConfig{
		Provider:            provider,
		BaseURL:             strings.TrimRight(baseURL, "/"),
		APIKey:              apiKey,
		Model:               model,
		SystemPrompt:        systemPrompt,
		Temperature:         temperature,
		TopP:                topP,
		TopK:                topK,
		PresencePenalty:     presencePenalty,
		RepetitionPenalty:   repetitionPenalty,
		MaxTokens:           maxTokens,
		ThinkingTokenBudget: thinkingTokenBudget,
		ReasoningEffort:     reasoningEffort,
		PreserveThinking:    preserveThinking,
		Timeout:             timeout,
	}, nil
}

func loadPass2Config(modelType string) (EndpointConfig, error) {
	provider, err := lookupRequiredString("PASS2_PROVIDER")
	if err != nil {
		return EndpointConfig{}, err
	}
	if err := validateProvider("PASS2_PROVIDER", provider); err != nil {
		return EndpointConfig{}, err
	}
	baseURL, err := lookupRequiredString("PASS2_BASE_URL")
	if err != nil {
		return EndpointConfig{}, err
	}
	apiKey, err := lookupRequiredString("PASS2_API_KEY")
	if err != nil {
		return EndpointConfig{}, err
	}
	model, err := lookupRequiredString("PASS2_MODEL")
	if err != nil {
		return EndpointConfig{}, err
	}
	systemPrompt := ""
	if modelType == ModelTypeQwenReasoning {
		systemPrompt, err = lookupRequiredString("PASS2_SYSTEM_PROMPT")
		if err != nil {
			return EndpointConfig{}, err
		}
	}
	temperature, err := lookupRequiredFloat("PASS2_TEMPERATURE")
	if err != nil {
		return EndpointConfig{}, err
	}
	topP, err := lookupOptionalFloat("PASS2_TOP_P")
	if err != nil {
		return EndpointConfig{}, err
	}
	topK, err := lookupOptionalInt("PASS2_TOP_K")
	if err != nil {
		return EndpointConfig{}, err
	}
	presencePenalty, err := lookupOptionalFloat("PASS2_PRESENCE_PENALTY")
	if err != nil {
		return EndpointConfig{}, err
	}
	repetitionPenalty, err := lookupOptionalFloat("PASS2_REPETITION_PENALTY")
	if err != nil {
		return EndpointConfig{}, err
	}
	maxTokens, err := lookupRequiredInt("PASS2_MAX_TOKENS")
	if err != nil {
		return EndpointConfig{}, err
	}
	if maxTokens <= 0 {
		return EndpointConfig{}, fmt.Errorf("PASS2_MAX_TOKENS must be > 0")
	}
	thinkingTokenBudget, err := lookupOptionalInt("PASS2_THINKING_TOKEN_BUDGET")
	if err != nil {
		return EndpointConfig{}, err
	}
	if thinkingTokenBudget != nil && *thinkingTokenBudget < 0 {
		return EndpointConfig{}, fmt.Errorf("PASS2_THINKING_TOKEN_BUDGET must be >= 0")
	}
	reasoningEffort, err := lookupOptionalString("PASS2_REASONING_EFFORT")
	if err != nil {
		return EndpointConfig{}, err
	}
	preserveThinking, err := lookupOptionalBool("PASS2_PRESERVE_THINKING")
	if err != nil {
		return EndpointConfig{}, err
	}
	timeout, err := lookupRequiredDuration("PASS2_TIMEOUT")
	if err != nil {
		return EndpointConfig{}, err
	}
	if timeout <= 0 {
		return EndpointConfig{}, fmt.Errorf("PASS2_TIMEOUT must be > 0")
	}

	return EndpointConfig{
		Provider:            provider,
		BaseURL:             strings.TrimRight(baseURL, "/"),
		APIKey:              apiKey,
		Model:               model,
		SystemPrompt:        systemPrompt,
		Temperature:         temperature,
		TopP:                topP,
		TopK:                topK,
		PresencePenalty:     presencePenalty,
		RepetitionPenalty:   repetitionPenalty,
		MaxTokens:           maxTokens,
		ThinkingTokenBudget: thinkingTokenBudget,
		ReasoningEffort:     reasoningEffort,
		PreserveThinking:    preserveThinking,
		Timeout:             timeout,
	}, nil
}

func validateModelType(value string) error {
	switch value {
	case ModelTypeQwenReasoning, ModelTypeGPTOss, ModelTypeGemma4:
		return nil
	default:
		return fmt.Errorf("MODEL_TYPE must be one of %q, %q, or %q", ModelTypeQwenReasoning, ModelTypeGPTOss, ModelTypeGemma4)
	}
}

func validateProvider(key, value string) error {
	switch value {
	case "vllm_chat_completions", "openai_chat_completions":
		return nil
	default:
		return fmt.Errorf("%s must be one of %q or %q", key, "vllm_chat_completions", "openai_chat_completions")
	}
}

func lookupRequiredString(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return value, nil
}

func lookupOptionalString(key string) (*string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil, nil
	}
	return &value, nil
}

func lookupRequiredInt(key string) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}
	return parsed, nil
}

func lookupRequiredFloat(key string) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid float: %w", key, err)
	}
	return parsed, nil
}

func lookupRequiredOptionalFloat(key string) (*float64, error) {
	value, err := lookupRequiredFloat(key)
	if err != nil {
		return nil, err
	}
	return float64Ptr(value), nil
}

func lookupOptionalFloat(key string) (*float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid float: %w", key, err)
	}
	return float64Ptr(parsed), nil
}

func lookupRequiredOptionalInt(key string) (*int, error) {
	value, err := lookupRequiredInt(key)
	if err != nil {
		return nil, err
	}
	return intPtr(value), nil
}

func lookupOptionalInt(key string) (*int, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}
	return intPtr(parsed), nil
}

func lookupRequiredOptionalBool(key string) (*bool, error) {
	value, err := lookupRequiredBool(key)
	if err != nil {
		return nil, err
	}
	return boolPtr(value), nil
}

func lookupRequiredBool(key string) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return false, fmt.Errorf("%s is required", key)
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a valid bool: %w", key, err)
	}
	return parsed, nil
}

func lookupOptionalBool(key string) (*bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid bool: %w", key, err)
	}
	return boolPtr(parsed), nil
}

func lookupRequiredDuration(key string) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return parsed, nil
}

func float64Ptr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

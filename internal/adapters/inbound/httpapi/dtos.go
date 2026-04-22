package httpapi

import "encoding/json"

type structuredRequestDTO struct {
	RequestID           string            `json:"request_id,omitempty"`
	Input               string            `json:"input"`
	SchemaVersion       string            `json:"schema_version,omitempty"`
	Schema              json.RawMessage   `json:"schema"`
	IncludeIntermediate *bool             `json:"include_intermediate,omitempty"`
	Pass1               *passOverridesDTO `json:"pass1,omitempty"`
	Pass2               *passOverridesDTO `json:"pass2,omitempty"`
	SinglePass          *passOverridesDTO `json:"single_pass,omitempty"`
}

type passOverridesDTO struct {
	Model               *string  `json:"model,omitempty"`
	SystemPrompt        *string  `json:"system_prompt,omitempty"`
	Temperature         *float64 `json:"temperature,omitempty"`
	TopP                *float64 `json:"top_p,omitempty"`
	TopK                *int     `json:"top_k,omitempty"`
	PresencePenalty     *float64 `json:"presence_penalty,omitempty"`
	RepetitionPenalty   *float64 `json:"repetition_penalty,omitempty"`
	MaxTokens           *int     `json:"max_tokens,omitempty"`
	ThinkingTokenBudget *int     `json:"thinking_token_budget,omitempty"`
	ReasoningEffort     *string  `json:"reasoning_effort,omitempty"`
	PreserveThinking    *bool    `json:"preserve_thinking,omitempty"`
}

type structuredResponseDTO struct {
	RequestID                  string              `json:"request_id"`
	IntermediateRepresentation string              `json:"intermediate_representation,omitempty"`
	Output                     json.RawMessage     `json:"output"`
	Metadata                   responseMetadataDTO `json:"metadata"`
}

type responseMetadataDTO struct {
	ExecutionMode           string          `json:"execution_mode"`
	SchemaVersion           string          `json:"schema_version,omitempty"`
	Pass1PromptVersion      string          `json:"pass1_prompt_version,omitempty"`
	Pass2PromptVersion      string          `json:"pass2_prompt_version,omitempty"`
	SinglePassPromptVersion string          `json:"single_pass_prompt_version,omitempty"`
	IRVersion               string          `json:"ir_version"`
	Pass1                   *passMetricsDTO `json:"pass1,omitempty"`
	Pass2                   *passMetricsDTO `json:"pass2,omitempty"`
	SinglePass              *passMetricsDTO `json:"single_pass,omitempty"`
}

type passMetricsDTO struct {
	Model                 string `json:"model"`
	Attempts              int    `json:"attempts"`
	LatencyMs             int64  `json:"latency_ms"`
	PromptTokens          int    `json:"prompt_tokens"`
	CompletionTokens      int    `json:"completion_tokens"`
	FinishReason          string `json:"finish_reason,omitempty"`
	ContentPresent        bool   `json:"content_present"`
	ReasoningPresent      bool   `json:"reasoning_present"`
	UsedReasoningFallback bool   `json:"used_reasoning_fallback"`
	Truncated             bool   `json:"truncated"`
}

type errorEnvelopeDTO struct {
	Error errorDTO `json:"error"`
}

type errorDTO struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

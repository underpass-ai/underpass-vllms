package httpapi

import "encoding/json"

type openAIResponsesCreateRequestDTO struct {
	Model           string                       `json:"model"`
	Input           json.RawMessage              `json:"input"`
	Instructions    json.RawMessage              `json:"instructions,omitempty"`
	Text            *openAIResponsesTextDTO      `json:"text,omitempty"`
	Temperature     *float64                     `json:"temperature,omitempty"`
	TopP            *float64                     `json:"top_p,omitempty"`
	MaxOutputTokens *int                         `json:"max_output_tokens,omitempty"`
	Reasoning       *openAIResponsesReasoningDTO `json:"reasoning,omitempty"`
	Stream          bool                         `json:"stream,omitempty"`
	Metadata        map[string]string            `json:"metadata,omitempty"`
}

type openAIResponsesReasoningDTO struct {
	Effort *string `json:"effort,omitempty"`
}

type openAIResponsesTextDTO struct {
	Format *openAIResponsesFormatDTO `json:"format,omitempty"`
}

type openAIResponsesFormatDTO struct {
	Type        string          `json:"type"`
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type openAIResponsesInputItemDTO struct {
	Type    string          `json:"type,omitempty"`
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content"`
}

type openAIResponsesInputContentPartDTO struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type openAIResponseDTO struct {
	ID                string                        `json:"id"`
	Object            string                        `json:"object"`
	CreatedAt         int64                         `json:"created_at"`
	CompletedAt       int64                         `json:"completed_at"`
	Status            string                        `json:"status"`
	Error             any                           `json:"error"`
	IncompleteDetails any                           `json:"incomplete_details"`
	Instructions      any                           `json:"instructions,omitempty"`
	MaxOutputTokens   *int                          `json:"max_output_tokens,omitempty"`
	Model             string                        `json:"model"`
	Output            []openAIResponseOutputItemDTO `json:"output"`
	OutputText        string                        `json:"output_text"`
	Reasoning         openAIResponseReasoningDTO    `json:"reasoning"`
	Text              openAIResponseTextDTO         `json:"text"`
	Usage             openAIResponseUsageDTO        `json:"usage"`
	Metadata          map[string]string             `json:"metadata"`
}

type openAIResponseOutputItemDTO struct {
	ID      string                           `json:"id"`
	Type    string                           `json:"type"`
	Status  string                           `json:"status"`
	Role    string                           `json:"role"`
	Content []openAIResponseOutputContentDTO `json:"content"`
}

type openAIResponseOutputContentDTO struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	Annotations []any  `json:"annotations"`
}

type openAIResponseReasoningDTO struct {
	Effort  *string `json:"effort"`
	Summary any     `json:"summary"`
}

type openAIResponseTextDTO struct {
	Format openAIResponsesFormatDTO `json:"format"`
}

type openAIResponseUsageDTO struct {
	InputTokens         int                                  `json:"input_tokens"`
	OutputTokens        int                                  `json:"output_tokens"`
	OutputTokensDetails openAIResponseOutputTokensDetailsDTO `json:"output_tokens_details"`
	TotalTokens         int                                  `json:"total_tokens"`
}

type openAIResponseOutputTokensDetailsDTO struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

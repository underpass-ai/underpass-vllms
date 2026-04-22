package httpapi

import "encoding/json"

type openAIChatCompletionRequestDTO struct {
	Model               string                   `json:"model"`
	Messages            []openAIMessageDTO       `json:"messages"`
	ResponseFormat      *openAIResponseFormatDTO `json:"response_format,omitempty"`
	Temperature         *float64                 `json:"temperature,omitempty"`
	TopP                *float64                 `json:"top_p,omitempty"`
	PresencePenalty     *float64                 `json:"presence_penalty,omitempty"`
	MaxCompletionTokens *int                     `json:"max_completion_tokens,omitempty"`
	MaxTokens           *int                     `json:"max_tokens,omitempty"`
	ReasoningEffort     *string                  `json:"reasoning_effort,omitempty"`
	Stream              bool                     `json:"stream,omitempty"`
	N                   *int                     `json:"n,omitempty"`
	User                string                   `json:"user,omitempty"`
	Metadata            map[string]string        `json:"metadata,omitempty"`
}

type openAIMessageDTO struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type openAIMessagePartDTO struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}

type openAIResponseFormatDTO struct {
	Type       string               `json:"type"`
	JSONSchema *openAIJSONSchemaDTO `json:"json_schema,omitempty"`
}

type openAIJSONSchemaDTO struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema"`
	Strict      *bool           `json:"strict,omitempty"`
}

type openAIChatCompletionResponseDTO struct {
	ID             string                          `json:"id"`
	Object         string                          `json:"object"`
	Created        int64                           `json:"created"`
	Model          string                          `json:"model"`
	RequestID      string                          `json:"request_id,omitempty"`
	Choices        []openAIChatCompletionChoiceDTO `json:"choices"`
	Usage          openAIUsageDTO                  `json:"usage"`
	ResponseFormat *openAIResponseFormatDTO        `json:"response_format,omitempty"`
}

type openAIChatCompletionChoiceDTO struct {
	Index        int                  `json:"index"`
	Message      openAIChatMessageDTO `json:"message"`
	FinishReason string               `json:"finish_reason,omitempty"`
}

type openAIChatMessageDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIUsageDTO struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIModelsListDTO struct {
	Object string               `json:"object"`
	Data   []openAIModelInfoDTO `json:"data"`
}

type openAIModelInfoDTO struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type openAIErrorEnvelopeDTO struct {
	Error openAIErrorDTO `json:"error"`
}

type openAIErrorDTO struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

type openAIRequestError struct {
	StatusCode int
	Payload    openAIErrorEnvelopeDTO
}

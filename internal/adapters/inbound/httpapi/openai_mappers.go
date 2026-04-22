package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

var permissiveJSONObjectSchema = json.RawMessage(`{"type":"object","additionalProperties":true}`)

func mapOpenAIChatCompletionRequestDTOToDomain(
	dto openAIChatCompletionRequestDTO,
	publicModel string,
) (domain.StructuredRequest, *openAIRequestError) {
	if err := validateOpenAIModel(dto.Model, publicModel); err != nil {
		return domain.StructuredRequest{}, err
	}
	if dto.Stream {
		return domain.StructuredRequest{}, newOpenAIInvalidRequest("stream", "stream=true is not supported")
	}
	if dto.N != nil && *dto.N != 1 {
		return domain.StructuredRequest{}, newOpenAIInvalidRequest("n", "only n=1 is supported")
	}

	input, err := flattenOpenAIMessages(dto.Messages)
	if err != nil {
		return domain.StructuredRequest{}, err
	}

	schema, schemaVersion, err := schemaFromOpenAIResponseFormat(dto.ResponseFormat)
	if err != nil {
		return domain.StructuredRequest{}, err
	}

	maxTokens, maxTokenErr := resolveOpenAIMaxTokens(dto.MaxCompletionTokens, dto.MaxTokens)
	if maxTokenErr != nil {
		return domain.StructuredRequest{}, maxTokenErr
	}

	modelName := domain.ModelName(dto.Model)
	includeIntermediate := false
	overrides := &domain.PassOverrides{
		Model:           &modelName,
		Temperature:     dto.Temperature,
		TopP:            dto.TopP,
		PresencePenalty: dto.PresencePenalty,
		MaxTokens:       maxTokens,
		ReasoningEffort: mapReasoningEffortPtr(dto.ReasoningEffort),
	}

	return domain.StructuredRequest{
		Input:               domain.InputPayload(input),
		SchemaVersion:       domain.SchemaVersion(schemaVersion),
		Schema:              schema,
		IncludeIntermediate: &includeIntermediate,
		Pass1:               overrides,
		Pass2:               overrides,
		SinglePass:          overrides,
	}, nil
}

func mapStructuredResponseDomainToOpenAIDTO(
	response domain.StructuredResponse,
	request openAIChatCompletionRequestDTO,
	publicModel string,
) openAIChatCompletionResponseDTO {
	model := resolveResponseModel(response.Metadata, publicModel)
	finishReason := resolveFinishReason(response.Metadata)
	usage := mapUsageFromMetadata(response.Metadata)

	return openAIChatCompletionResponseDTO{
		ID:        "chatcmpl-" + string(response.RequestID),
		Object:    "chat.completion",
		Created:   time.Now().Unix(),
		Model:     model,
		RequestID: string(response.RequestID),
		Choices: []openAIChatCompletionChoiceDTO{
			{
				Index: 0,
				Message: openAIChatMessageDTO{
					Role:    "assistant",
					Content: strings.TrimSpace(string(response.Output)),
				},
				FinishReason: finishReason,
			},
		},
		Usage:          usage,
		ResponseFormat: request.ResponseFormat,
	}
}

func mapDomainErrorToOpenAIDTO(err *domain.Error) openAIErrorEnvelopeDTO {
	errorType := "api_error"
	if err.StatusCode >= 400 && err.StatusCode < 500 {
		errorType = "invalid_request_error"
	}

	return openAIErrorEnvelopeDTO{
		Error: openAIErrorDTO{
			Message: err.Message,
			Type:    errorType,
			Code:    string(err.Code),
		},
	}
}

func buildOpenAIModelsListDTO(publicModel string) openAIModelsListDTO {
	now := time.Now().Unix()
	return openAIModelsListDTO{
		Object: "list",
		Data: []openAIModelInfoDTO{
			{
				ID:      publicModel,
				Object:  "model",
				Created: now,
				OwnedBy: "underpassai",
			},
		},
	}
}

func buildOpenAIModelInfoDTO(publicModel string) openAIModelInfoDTO {
	return openAIModelInfoDTO{
		ID:      publicModel,
		Object:  "model",
		Created: time.Now().Unix(),
		OwnedBy: "underpassai",
	}
}

func validateOpenAIModel(requestedModel string, publicModel string) *openAIRequestError {
	if strings.TrimSpace(requestedModel) == "" {
		return newOpenAIInvalidRequest("model", "model is required")
	}
	if strings.TrimSpace(publicModel) == "" {
		return nil
	}
	if requestedModel != publicModel {
		return &openAIRequestError{
			StatusCode: 400,
			Payload: openAIErrorEnvelopeDTO{
				Error: openAIErrorDTO{
					Message: fmt.Sprintf("unsupported model %q", requestedModel),
					Type:    "invalid_request_error",
					Param:   "model",
					Code:    "model_not_found",
				},
			},
		}
	}
	return nil
}

func flattenOpenAIMessages(messages []openAIMessageDTO) (string, *openAIRequestError) {
	if len(messages) == 0 {
		return "", newOpenAIInvalidRequest("messages", "messages must not be empty")
	}

	var builder strings.Builder
	for index, message := range messages {
		if !isSupportedOpenAIRole(message.Role) {
			return "", newOpenAIInvalidRequest("messages", fmt.Sprintf("unsupported message role %q", message.Role))
		}

		content, err := flattenOpenAIMessageContent(message.Content)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(content) == "" {
			return "", newOpenAIInvalidRequest("messages", fmt.Sprintf("message %d content must not be empty", index))
		}

		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(strings.ToUpper(message.Role))
		builder.WriteString(":\n")
		builder.WriteString(content)
	}

	return builder.String(), nil
}

func isSupportedOpenAIRole(role string) bool {
	switch role {
	case "system", "developer", "user", "assistant":
		return true
	default:
		return false
	}
}

func flattenOpenAIMessageContent(raw json.RawMessage) (string, *openAIRequestError) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text, nil
	}

	var parts []openAIMessagePartDTO
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", newOpenAIInvalidRequest("messages", "message content must be a string or array of text parts")
	}

	var builder strings.Builder
	for _, part := range parts {
		switch part.Type {
		case "text":
			builder.WriteString(part.Text)
		case "refusal":
			builder.WriteString(part.Refusal)
		default:
			return "", newOpenAIInvalidRequest("messages", fmt.Sprintf("unsupported message content part type %q", part.Type))
		}
	}

	return builder.String(), nil
}

func schemaFromOpenAIResponseFormat(format *openAIResponseFormatDTO) (json.RawMessage, string, *openAIRequestError) {
	if format == nil {
		return nil, "", newOpenAIInvalidRequest("response_format", "response_format is required")
	}

	switch format.Type {
	case "json_schema":
		if format.JSONSchema == nil {
			return nil, "", newOpenAIInvalidRequest("response_format", "response_format.json_schema is required")
		}
		if strings.TrimSpace(format.JSONSchema.Name) == "" {
			return nil, "", newOpenAIInvalidRequest("response_format", "response_format.json_schema.name is required")
		}
		if len(format.JSONSchema.Schema) == 0 || !json.Valid(format.JSONSchema.Schema) {
			return nil, "", newOpenAIInvalidRequest("response_format", "response_format.json_schema.schema must be valid JSON")
		}
		return format.JSONSchema.Schema, format.JSONSchema.Name, nil
	case "json_object":
		return permissiveJSONObjectSchema, "json_object", nil
	default:
		return nil, "", newOpenAIInvalidRequest("response_format", fmt.Sprintf("unsupported response_format type %q", format.Type))
	}
}

func resolveOpenAIMaxTokens(maxCompletionTokens *int, maxTokens *int) (*int, *openAIRequestError) {
	if maxCompletionTokens != nil && maxTokens != nil && *maxCompletionTokens != *maxTokens {
		return nil, newOpenAIInvalidRequest("max_completion_tokens", "max_completion_tokens and max_tokens must match when both are provided")
	}
	if maxCompletionTokens != nil {
		return maxCompletionTokens, nil
	}
	if maxTokens != nil {
		return maxTokens, nil
	}
	return nil, nil
}

func resolveResponseModel(metadata domain.ResponseMetadata, fallback string) string {
	switch metadata.ExecutionMode {
	case domain.ExecutionModeSinglePass:
		if metadata.SinglePass.Model != "" {
			return string(metadata.SinglePass.Model)
		}
	case domain.ExecutionModeTwoPass:
		if metadata.Pass2.Model != "" {
			return string(metadata.Pass2.Model)
		}
		if metadata.Pass1.Model != "" {
			return string(metadata.Pass1.Model)
		}
	}
	return fallback
}

func resolveFinishReason(metadata domain.ResponseMetadata) string {
	switch metadata.ExecutionMode {
	case domain.ExecutionModeSinglePass:
		return metadata.SinglePass.FinishReason
	case domain.ExecutionModeTwoPass:
		return metadata.Pass2.FinishReason
	default:
		return ""
	}
}

func mapUsageFromMetadata(metadata domain.ResponseMetadata) openAIUsageDTO {
	switch metadata.ExecutionMode {
	case domain.ExecutionModeSinglePass:
		total := metadata.SinglePass.PromptTokens + metadata.SinglePass.CompletionTokens
		return openAIUsageDTO{
			PromptTokens:     metadata.SinglePass.PromptTokens,
			CompletionTokens: metadata.SinglePass.CompletionTokens,
			TotalTokens:      total,
		}
	case domain.ExecutionModeTwoPass:
		promptTokens := metadata.Pass1.PromptTokens + metadata.Pass2.PromptTokens
		completionTokens := metadata.Pass1.CompletionTokens + metadata.Pass2.CompletionTokens
		return openAIUsageDTO{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		}
	default:
		return openAIUsageDTO{}
	}
}

func newOpenAIInvalidRequest(param string, message string) *openAIRequestError {
	return &openAIRequestError{
		StatusCode: 400,
		Payload: openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: message,
				Type:    "invalid_request_error",
				Param:   param,
				Code:    "invalid_request_error",
			},
		},
	}
}

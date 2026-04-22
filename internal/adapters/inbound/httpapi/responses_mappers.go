package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func mapOpenAIResponsesRequestDTOToDomain(
	dto openAIResponsesCreateRequestDTO,
	publicModel string,
) (domain.StructuredRequest, *openAIRequestError) {
	if err := validateOpenAIModel(dto.Model, publicModel); err != nil {
		return domain.StructuredRequest{}, err
	}
	if dto.Stream {
		return domain.StructuredRequest{}, newOpenAIInvalidRequest("stream", "stream=true is not supported")
	}

	input, err := flattenResponsesInput(dto.Input)
	if err != nil {
		return domain.StructuredRequest{}, err
	}

	instructions, err := flattenResponsesInstructions(dto.Instructions)
	if err != nil {
		return domain.StructuredRequest{}, err
	}

	schema, schemaVersion, err := schemaFromResponsesTextFormat(dto.Text)
	if err != nil {
		return domain.StructuredRequest{}, err
	}

	fullInput := input
	if instructions != "" {
		fullInput = "DEVELOPER:\n" + instructions + "\n\n" + fullInput
	}

	modelName := domain.ModelName(dto.Model)
	includeIntermediate := false
	overrides := &domain.PassOverrides{
		Model:           &modelName,
		Temperature:     dto.Temperature,
		TopP:            dto.TopP,
		MaxTokens:       dto.MaxOutputTokens,
		ReasoningEffort: mapResponsesReasoningEffortPtr(dto.Reasoning),
	}

	return domain.StructuredRequest{
		Input:               domain.InputPayload(fullInput),
		SchemaVersion:       domain.SchemaVersion(schemaVersion),
		Schema:              schema,
		IncludeIntermediate: &includeIntermediate,
		Pass1:               overrides,
		Pass2:               overrides,
		SinglePass:          overrides,
	}, nil
}

func mapStructuredResponseDomainToResponsesDTO(
	response domain.StructuredResponse,
	request openAIResponsesCreateRequestDTO,
	publicModel string,
) openAIResponseDTO {
	now := time.Now().Unix()
	model := resolveResponseModel(response.Metadata, publicModel)
	outputText := strings.TrimSpace(string(response.Output))
	reasoningEffort := resolveResponsesReasoningEffort(request)

	return openAIResponseDTO{
		ID:                "resp_" + string(response.RequestID),
		Object:            "response",
		CreatedAt:         now,
		CompletedAt:       now,
		Status:            "completed",
		Error:             nil,
		IncompleteDetails: nil,
		Instructions:      normalizeResponsesInstructions(request.Instructions),
		MaxOutputTokens:   request.MaxOutputTokens,
		Model:             model,
		Output: []openAIResponseOutputItemDTO{
			{
				ID:     "msg_" + string(response.RequestID),
				Type:   "message",
				Status: "completed",
				Role:   "assistant",
				Content: []openAIResponseOutputContentDTO{
					{
						Type:        "output_text",
						Text:        outputText,
						Annotations: []any{},
					},
				},
			},
		},
		OutputText: outputText,
		Reasoning: openAIResponseReasoningDTO{
			Effort:  reasoningEffort,
			Summary: nil,
		},
		Text: openAIResponseTextDTO{
			Format: normalizeResponsesFormat(request.Text),
		},
		Usage:    mapUsageFromMetadataToResponses(response.Metadata),
		Metadata: map[string]string{},
	}
}

func flattenResponsesInput(raw json.RawMessage) (string, *openAIRequestError) {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if strings.TrimSpace(text) == "" {
			return "", newOpenAIInvalidRequest("input", "input must not be empty")
		}
		return "USER:\n" + text, nil
	}

	var items []openAIResponsesInputItemDTO
	if err := json.Unmarshal(raw, &items); err != nil {
		return "", newOpenAIInvalidRequest("input", "input must be a string or an array of message items")
	}
	if len(items) == 0 {
		return "", newOpenAIInvalidRequest("input", "input must not be empty")
	}

	var messages []openAIMessageDTO
	for _, item := range items {
		itemType := strings.TrimSpace(item.Type)
		if itemType != "" && itemType != "message" {
			return "", newOpenAIInvalidRequest("input", fmt.Sprintf("unsupported input item type %q", item.Type))
		}
		role := item.Role
		if strings.TrimSpace(role) == "" {
			role = "user"
		}
		messages = append(messages, openAIMessageDTO{
			Role:    role,
			Content: item.Content,
		})
	}

	return flattenOpenAIMessages(messages)
}

func flattenResponsesInstructions(raw json.RawMessage) (string, *openAIRequestError) {
	if len(raw) == 0 {
		return "", nil
	}
	return flattenOpenAIMessageContent(raw)
}

func schemaFromResponsesTextFormat(text *openAIResponsesTextDTO) (json.RawMessage, string, *openAIRequestError) {
	if text == nil || text.Format == nil {
		return nil, "", newOpenAIInvalidRequest("text.format", "text.format is required")
	}

	switch text.Format.Type {
	case "json_schema":
		if strings.TrimSpace(text.Format.Name) == "" {
			return nil, "", newOpenAIInvalidRequest("text.format.name", "text.format.name is required")
		}
		if len(text.Format.Schema) == 0 || !json.Valid(text.Format.Schema) {
			return nil, "", newOpenAIInvalidRequest("text.format.schema", "text.format.schema must be valid JSON")
		}
		return text.Format.Schema, text.Format.Name, nil
	case "json_object":
		return permissiveJSONObjectSchema, "json_object", nil
	case "text":
		return nil, "", newOpenAIInvalidRequest("text.format.type", "text.format.type=text is not supported by this orchestrator")
	default:
		return nil, "", newOpenAIInvalidRequest("text.format.type", fmt.Sprintf("unsupported text.format.type %q", text.Format.Type))
	}
}

func normalizeResponsesInstructions(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	return json.RawMessage(raw)
}

func normalizeResponsesFormat(text *openAIResponsesTextDTO) openAIResponsesFormatDTO {
	if text == nil || text.Format == nil {
		return openAIResponsesFormatDTO{Type: "json_schema", Name: "structured_output", Schema: permissiveJSONObjectSchema}
	}
	return *text.Format
}

func mapUsageFromMetadataToResponses(metadata domain.ResponseMetadata) openAIResponseUsageDTO {
	usage := mapUsageFromMetadata(metadata)
	return openAIResponseUsageDTO{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		OutputTokensDetails: openAIResponseOutputTokensDetailsDTO{
			ReasoningTokens: 0,
		},
		TotalTokens: usage.TotalTokens,
	}
}

func resolveResponsesReasoningEffort(request openAIResponsesCreateRequestDTO) *string {
	if request.Reasoning == nil {
		return nil
	}
	return request.Reasoning.Effort
}

func mapResponsesReasoningEffortPtr(value *openAIResponsesReasoningDTO) *domain.ReasoningEffort {
	if value == nil || value.Effort == nil {
		return nil
	}
	mapped := domain.ReasoningEffort(*value.Effort)
	return &mapped
}

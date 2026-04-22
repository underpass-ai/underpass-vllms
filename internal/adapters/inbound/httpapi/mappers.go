package httpapi

import (
	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func mapStructuredRequestDTOToDomain(dto structuredRequestDTO) domain.StructuredRequest {
	return domain.StructuredRequest{
		RequestID:           domain.RequestID(dto.RequestID),
		Input:               domain.InputPayload(dto.Input),
		SchemaVersion:       domain.SchemaVersion(dto.SchemaVersion),
		Schema:              dto.Schema,
		IncludeIntermediate: dto.IncludeIntermediate,
		Pass1:               mapPassOverridesDTOToDomain(dto.Pass1),
		Pass2:               mapPassOverridesDTOToDomain(dto.Pass2),
		SinglePass:          mapPassOverridesDTOToDomain(dto.SinglePass),
	}
}

func mapPassOverridesDTOToDomain(dto *passOverridesDTO) *domain.PassOverrides {
	if dto == nil {
		return nil
	}
	return &domain.PassOverrides{
		Model:               mapModelNamePtr(dto.Model),
		SystemPrompt:        dto.SystemPrompt,
		Temperature:         dto.Temperature,
		TopP:                dto.TopP,
		TopK:                dto.TopK,
		PresencePenalty:     dto.PresencePenalty,
		RepetitionPenalty:   dto.RepetitionPenalty,
		MaxTokens:           dto.MaxTokens,
		ThinkingTokenBudget: dto.ThinkingTokenBudget,
		ReasoningEffort:     mapReasoningEffortPtr(dto.ReasoningEffort),
		PreserveThinking:    dto.PreserveThinking,
	}
}

func mapStructuredResponseDomainToDTO(response domain.StructuredResponse) structuredResponseDTO {
	return structuredResponseDTO{
		RequestID:                  string(response.RequestID),
		IntermediateRepresentation: response.IntermediateRepresentation,
		Output:                     response.Output,
		Metadata:                   mapResponseMetadataDomainToDTO(response.Metadata),
	}
}

func mapResponseMetadataDomainToDTO(metadata domain.ResponseMetadata) responseMetadataDTO {
	dto := responseMetadataDTO{
		ExecutionMode: string(metadata.ExecutionMode),
		SchemaVersion: string(metadata.SchemaVersion),
		IRVersion:     string(metadata.IRVersion),
	}

	switch metadata.ExecutionMode {
	case domain.ExecutionModeTwoPass:
		dto.Pass1PromptVersion = string(metadata.Pass1PromptVersion)
		dto.Pass2PromptVersion = string(metadata.Pass2PromptVersion)
		pass1 := mapPassMetricsDomainToDTO(metadata.Pass1)
		pass2 := mapPassMetricsDomainToDTO(metadata.Pass2)
		dto.Pass1 = &pass1
		dto.Pass2 = &pass2
	case domain.ExecutionModeSinglePass:
		dto.SinglePassPromptVersion = string(metadata.SinglePassPromptVersion)
		singlePass := mapPassMetricsDomainToDTO(metadata.SinglePass)
		dto.SinglePass = &singlePass
	}

	return dto
}

func mapPassMetricsDomainToDTO(metrics domain.PassMetrics) passMetricsDTO {
	return passMetricsDTO{
		Model:                 string(metrics.Model),
		Attempts:              metrics.Attempts,
		LatencyMs:             metrics.LatencyMs,
		PromptTokens:          metrics.PromptTokens,
		CompletionTokens:      metrics.CompletionTokens,
		FinishReason:          metrics.FinishReason,
		ContentPresent:        metrics.ContentPresent,
		ReasoningPresent:      metrics.ReasoningPresent,
		UsedReasoningFallback: metrics.UsedReasoningFallback,
		Truncated:             metrics.Truncated,
	}
}

func mapDomainErrorToDTO(err *domain.Error) errorEnvelopeDTO {
	return errorEnvelopeDTO{
		Error: errorDTO{
			Code:      string(err.Code),
			Message:   err.Message,
			Retryable: err.Retryable,
			Details:   err.Details,
		},
	}
}

func mapModelNamePtr(value *string) *domain.ModelName {
	if value == nil {
		return nil
	}
	mapped := domain.ModelName(*value)
	return &mapped
}

func mapReasoningEffortPtr(value *string) *domain.ReasoningEffort {
	if value == nil {
		return nil
	}
	mapped := domain.ReasoningEffort(*value)
	return &mapped
}

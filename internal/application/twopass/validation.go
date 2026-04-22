package twopassapp

import (
	"encoding/json"
	"strings"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func validateRequest(request domain.StructuredRequest) *domain.Error {
	if strings.TrimSpace(string(request.Input)) == "" {
		return &domain.Error{
			StatusCode: 400,
			Code:       domain.ErrorCodeInvalidRequest,
			Message:    "input is required",
		}
	}
	if len(request.Schema) == 0 || !json.Valid(request.Schema) {
		return &domain.Error{
			StatusCode: 400,
			Code:       domain.ErrorCodeInvalidRequest,
			Message:    "schema must be valid JSON",
		}
	}
	return nil
}

func validateIntermediateSize(intermediate domain.IntermediateRepresentation, maxBytes int) *domain.Error {
	if intermediate.SizeBytes() <= maxBytes {
		return nil
	}
	return &domain.Error{
		StatusCode: 502,
		Code:       domain.ErrorCodePass1TooLarge,
		Message:    "Pass 1 intermediate representation exceeded the configured limit",
	}
}

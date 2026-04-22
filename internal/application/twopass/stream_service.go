package twopassapp

import (
	"context"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type StreamService struct {
	executor domain.StructuredStreamExecutionPort
}

func NewStreamService(executor domain.StructuredStreamExecutionPort) *StreamService {
	return &StreamService{executor: executor}
}

func (s *StreamService) Stream(
	ctx context.Context,
	request domain.StructuredRequest,
	emit func(domain.CompletionDelta) error,
) (domain.StructuredResponse, *domain.Error) {
	if err := validateRequest(request); err != nil {
		return domain.StructuredResponse{}, err
	}

	requestID := request.RequestID
	if requestID == "" {
		requestID = newRequestID()
	}

	result, err := s.executor.Stream(ctx, requestID, request, emit)
	if err != nil {
		return domain.StructuredResponse{}, err
	}

	result.Metadata.SchemaVersion = request.SchemaVersion

	return domain.StructuredResponse{
		RequestID: requestID,
		Output:    result.Output,
		Metadata:  result.Metadata,
	}, nil
}

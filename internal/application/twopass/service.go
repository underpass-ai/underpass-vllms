package twopassapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type PassDefaults struct {
	Model               domain.ModelName
	SystemPrompt        string
	Temperature         float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	RepetitionPenalty   *float64
	MaxTokens           int
	ThinkingTokenBudget *int
	ReasoningEffort     *domain.ReasoningEffort
	PreserveThinking    *bool
}

type Versions struct {
	Pass1Prompt      domain.PromptVersion
	Pass2Prompt      domain.PromptVersion
	SinglePassPrompt domain.PromptVersion
	IR               domain.IRVersion
}

type PromptTemplates struct {
	Pass1User      string
	Pass2User      string
	Pass2RetryHint string
	SinglePassUser string
}

type Settings struct {
	MaxIntermediateBytes   int
	Pass2RetryCount        int
	Versions               Versions
	PromptTemplates        PromptTemplates
	SinglePassSystemPrompt string
	Pass1                  PassDefaults
	Pass2                  PassDefaults
	SinglePass             PassDefaults
}

type Service struct {
	executor domain.StructuredExecutionPort
}

func NewService(executor domain.StructuredExecutionPort) *Service {
	return &Service{executor: executor}
}

func (s *Service) Execute(ctx context.Context, request domain.StructuredRequest) (domain.StructuredResponse, *domain.Error) {
	if err := validateRequest(request); err != nil {
		return domain.StructuredResponse{}, err
	}

	requestID := request.RequestID
	if requestID == "" {
		requestID = newRequestID()
	}

	result, err := s.executor.Execute(ctx, requestID, request)
	if err != nil {
		return domain.StructuredResponse{}, err
	}

	result.Metadata.SchemaVersion = request.SchemaVersion

	response := domain.StructuredResponse{
		RequestID: requestID,
		Output:    result.Output,
		Metadata:  result.Metadata,
	}
	if wantsIntermediate(request.IncludeIntermediate) {
		response.IntermediateRepresentation = result.IntermediateRepresentation
	}

	return response, nil
}

func wantsIntermediate(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

func transportError(code domain.ErrorCode, message string, err error) *domain.Error {
	return &domain.Error{
		StatusCode: 502,
		Code:       code,
		Message:    message,
		Details: map[string]string{
			"cause": err.Error(),
		},
	}
}

func newRequestID() domain.RequestID {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return domain.RequestID(fmt.Sprintf("req-%d", time.Now().UnixNano()))
	}
	return domain.RequestID(hex.EncodeToString(buffer))
}

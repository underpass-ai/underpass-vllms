package twopass

import (
	"context"
	"encoding/json"
)

type RequestID string
type SchemaVersion string
type PromptVersion string
type IRVersion string
type ModelName string
type InputPayload string
type IntermediateRepresentation string
type ExecutionMode string
type ReasoningEffort string
type Role string

const (
	ExecutionModeTwoPass    ExecutionMode = "two_pass"
	ExecutionModeSinglePass ExecutionMode = "single_pass"

	SystemRole Role = "system"
	UserRole   Role = "user"
)

type Message struct {
	Role    Role
	Content string
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type CompletionRequest struct {
	Model               ModelName
	Messages            []Message
	Temperature         float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	RepetitionPenalty   *float64
	MaxTokens           int
	ThinkingTokenBudget *int
	StructuredSchema    json.RawMessage
	ReasoningEffort     *ReasoningEffort
	PreserveThinking    *bool
	DisableThinking     bool
}

type CompletionResponse struct {
	Content      string
	Reasoning    string
	FinishReason string
	Usage        Usage
}

type CompletionPort interface {
	Complete(ctx context.Context, request CompletionRequest) (CompletionResponse, error)
}

type CompletionDelta struct {
	Content      string
	Reasoning    string
	FinishReason string
	Usage        *Usage
}

type CompletionStreamPort interface {
	Stream(ctx context.Context, request CompletionRequest, emit func(CompletionDelta) error) (CompletionResponse, error)
}

type SchemaValidatorPort interface {
	Validate(schema json.RawMessage, candidate json.RawMessage) error
}

type StructuredExecutionResult struct {
	IntermediateRepresentation string
	Output                     json.RawMessage
	Metadata                   ResponseMetadata
}

type StructuredExecutionPort interface {
	Execute(ctx context.Context, requestID RequestID, request StructuredRequest) (StructuredExecutionResult, *Error)
}

type StructuredStreamExecutionPort interface {
	Stream(ctx context.Context, requestID RequestID, request StructuredRequest, emit func(CompletionDelta) error) (StructuredExecutionResult, *Error)
}

type PassOverrides struct {
	Model               *ModelName
	SystemPrompt        *string
	Temperature         *float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	RepetitionPenalty   *float64
	MaxTokens           *int
	ThinkingTokenBudget *int
	ReasoningEffort     *ReasoningEffort
	PreserveThinking    *bool
}

type StructuredRequest struct {
	RequestID           RequestID
	Input               InputPayload
	SchemaVersion       SchemaVersion
	Schema              json.RawMessage
	IncludeIntermediate *bool
	Pass1               *PassOverrides
	Pass2               *PassOverrides
	SinglePass          *PassOverrides
}

type PassMetrics struct {
	Model                 ModelName
	Attempts              int
	LatencyMs             int64
	PromptTokens          int
	CompletionTokens      int
	FinishReason          string
	ContentPresent        bool
	ReasoningPresent      bool
	UsedReasoningFallback bool
	Truncated             bool
}

type ResponseMetadata struct {
	ExecutionMode           ExecutionMode
	SchemaVersion           SchemaVersion
	Pass1PromptVersion      PromptVersion
	Pass2PromptVersion      PromptVersion
	SinglePassPromptVersion PromptVersion
	IRVersion               IRVersion
	Pass1                   PassMetrics
	Pass2                   PassMetrics
	SinglePass              PassMetrics
}

type StructuredResponse struct {
	RequestID                  RequestID
	IntermediateRepresentation string
	Output                     json.RawMessage
	Metadata                   ResponseMetadata
}

func (ir IntermediateRepresentation) SizeBytes() int {
	return len([]byte(ir))
}

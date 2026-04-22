package twopass

type ErrorCode string

const (
	ErrorCodeInvalidRequest      ErrorCode = "invalid_request"
	ErrorCodePass1Transport      ErrorCode = "pass1_transport_failure"
	ErrorCodePass1Empty          ErrorCode = "pass1_empty_response"
	ErrorCodePass1TooLarge       ErrorCode = "pass1_ir_too_large"
	ErrorCodePass2Transport      ErrorCode = "pass2_transport_failure"
	ErrorCodePass2Empty          ErrorCode = "pass2_empty_response"
	ErrorCodePass2Validation     ErrorCode = "pass2_validation_failure"
	ErrorCodePass2Exhausted      ErrorCode = "pass2_exhausted"
	ErrorCodeUnsupportedResponse ErrorCode = "unsupported_model_response"
)

type Error struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	Retryable  bool
	Details    map[string]string
}

func (e *Error) Error() string {
	return e.Message
}

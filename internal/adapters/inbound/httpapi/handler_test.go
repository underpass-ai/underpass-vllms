package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type useCaseStub struct {
	response domain.StructuredResponse
	err      *domain.Error
}

func (s *useCaseStub) Execute(_ context.Context, _ domain.StructuredRequest) (domain.StructuredResponse, *domain.Error) {
	return s.response, s.err
}

func TestStructuredReturnsUseCaseResponse(t *testing.T) {
	handler := NewHandler(&useCaseStub{
		response: domain.StructuredResponse{
			RequestID: "req-1",
			Output:    json.RawMessage(`{"ok":true}`),
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/two-pass/structured", strings.NewReader(`{"input":"hello","schema":{"type":"object"}}`))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"request_id":"req-1"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

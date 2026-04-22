package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type UseCase interface {
	Execute(ctx context.Context, request domain.StructuredRequest) (domain.StructuredResponse, *domain.Error)
}

type Handler struct {
	useCase UseCase
}

func NewHandler(useCase UseCase) http.Handler {
	handler := &Handler{useCase: useCase}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.healthz)
	mux.HandleFunc("/readyz", handler.healthz)
	mux.HandleFunc("/v1/two-pass/structured", handler.structured)
	return mux
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) structured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": map[string]any{
				"code":    "method_not_allowed",
				"message": "only POST is supported",
			},
		})
		return
	}

	defer r.Body.Close()

	var requestDTO structuredRequestDTO
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&requestDTO); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelopeDTO{
			Error: errorDTO{
				Code:    string(domain.ErrorCodeInvalidRequest),
				Message: err.Error(),
			},
		})
		return
	}

	request := mapStructuredRequestDTOToDomain(requestDTO)
	response, execErr := h.useCase.Execute(r.Context(), request)
	if execErr != nil {
		writeJSON(w, execErr.StatusCode, mapDomainErrorToDTO(execErr))
		return
	}

	writeJSON(w, http.StatusOK, mapStructuredResponseDomainToDTO(response))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

type UseCase interface {
	Execute(ctx context.Context, request domain.StructuredRequest) (domain.StructuredResponse, *domain.Error)
}

type Handler struct {
	useCase     UseCase
	publicModel string
}

type Option func(*Handler)

func WithPublicModel(model string) Option {
	return func(handler *Handler) {
		handler.publicModel = model
	}
}

func NewHandler(useCase UseCase, options ...Option) http.Handler {
	handler := &Handler{useCase: useCase}
	for _, option := range options {
		option(handler)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.healthz)
	mux.HandleFunc("/readyz", handler.healthz)
	mux.HandleFunc("/v1/models", handler.models)
	mux.HandleFunc("/v1/models/", handler.modelByID)
	mux.HandleFunc("/v1/chat/completions", handler.chatCompletions)
	mux.HandleFunc("/v1/responses", handler.responses)
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

func (h *Handler) models(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": map[string]any{
				"code":    "method_not_allowed",
				"message": "only GET is supported",
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, buildOpenAIModelsListDTO(h.publicModel))
}

func (h *Handler) modelByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": map[string]any{
				"code":    "method_not_allowed",
				"message": "only GET is supported",
			},
		})
		return
	}

	modelID := strings.TrimPrefix(r.URL.Path, "/v1/models/")
	if modelID == "" || modelID != h.publicModel {
		writeJSON(w, http.StatusNotFound, openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: "model not found",
				Type:    "invalid_request_error",
				Param:   "model",
				Code:    "model_not_found",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, buildOpenAIModelInfoDTO(h.publicModel))
}

func (h *Handler) chatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: "only POST is supported",
				Type:    "invalid_request_error",
				Code:    "method_not_allowed",
			},
		})
		return
	}

	defer r.Body.Close()

	var requestDTO openAIChatCompletionRequestDTO
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestDTO); err != nil {
		writeJSON(w, http.StatusBadRequest, openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_request_error",
			},
		})
		return
	}

	request, mapErr := mapOpenAIChatCompletionRequestDTOToDomain(requestDTO, h.publicModel)
	if mapErr != nil {
		writeJSON(w, mapErr.StatusCode, mapErr.Payload)
		return
	}

	response, execErr := h.useCase.Execute(r.Context(), request)
	if execErr != nil {
		writeJSON(w, execErr.StatusCode, mapDomainErrorToOpenAIDTO(execErr))
		return
	}

	writeJSON(w, http.StatusOK, mapStructuredResponseDomainToOpenAIDTO(response, requestDTO, h.publicModel))
}

func (h *Handler) responses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: "only POST is supported",
				Type:    "invalid_request_error",
				Code:    "method_not_allowed",
			},
		})
		return
	}

	defer r.Body.Close()

	var requestDTO openAIResponsesCreateRequestDTO
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestDTO); err != nil {
		writeJSON(w, http.StatusBadRequest, openAIErrorEnvelopeDTO{
			Error: openAIErrorDTO{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_request_error",
			},
		})
		return
	}

	request, mapErr := mapOpenAIResponsesRequestDTOToDomain(requestDTO, h.publicModel)
	if mapErr != nil {
		writeJSON(w, mapErr.StatusCode, mapErr.Payload)
		return
	}

	response, execErr := h.useCase.Execute(r.Context(), request)
	if execErr != nil {
		writeJSON(w, execErr.StatusCode, mapDomainErrorToOpenAIDTO(execErr))
		return
	}

	writeJSON(w, http.StatusOK, mapStructuredResponseDomainToResponsesDTO(response, requestDTO, h.publicModel))
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

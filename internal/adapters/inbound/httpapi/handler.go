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

type StreamUseCase interface {
	Stream(ctx context.Context, request domain.StructuredRequest, emit func(domain.CompletionDelta) error) (domain.StructuredResponse, *domain.Error)
}

type Handler struct {
	useCase     UseCase
	streamer    StreamUseCase
	publicModel string
}

type Option func(*Handler)

func WithPublicModel(model string) Option {
	return func(handler *Handler) {
		handler.publicModel = model
	}
}

func WithStreamer(streamer StreamUseCase) Option {
	return func(handler *Handler) {
		handler.streamer = streamer
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
	if requestDTO.Stream {
		h.streamChatCompletions(w, r, requestDTO, request)
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
	if requestDTO.Stream {
		h.streamResponses(w, r, requestDTO, request)
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

func (h *Handler) streamChatCompletions(
	w http.ResponseWriter,
	r *http.Request,
	requestDTO openAIChatCompletionRequestDTO,
	request domain.StructuredRequest,
) {
	if h.streamer == nil {
		writeJSON(w, http.StatusBadRequest, newOpenAIInvalidRequest("stream", "stream=true is only supported for single_pass backends").Payload)
		return
	}

	stream := newSSEWriter(w)
	model := requestDTO.Model
	if request.RequestID == "" {
		request.RequestID = domain.RequestID(newStreamingRequestID())
	}
	started := false

	start := func(requestID string) error {
		if started {
			return nil
		}
		if err := stream.Start(); err != nil {
			return err
		}
		if err := stream.Data(mapStructuredStreamStartToOpenAIChunkDTO(requestID, model)); err != nil {
			return err
		}
		started = true
		return nil
	}

	response, execErr := h.streamer.Stream(r.Context(), request, func(delta domain.CompletionDelta) error {
		if strings.TrimSpace(delta.Content) == "" {
			return nil
		}
		requestID := string(request.RequestID)
		if err := start(requestID); err != nil {
			return err
		}
		return stream.Data(mapStructuredStreamDeltaToOpenAIChunkDTO(requestID, model, delta.Content))
	})
	if execErr != nil {
		if !started {
			writeJSON(w, execErr.StatusCode, mapDomainErrorToOpenAIDTO(execErr))
			return
		}
		_ = stream.Data(mapDomainErrorToOpenAIDTO(execErr))
		_ = stream.Done()
		return
	}

	requestID := string(response.RequestID)
	if err := start(requestID); err != nil {
		return
	}
	if err := stream.Data(mapStructuredStreamDoneToOpenAIChunkDTO(requestID, model, resolveFinishReason(response.Metadata))); err != nil {
		return
	}
	_ = stream.Done()
}

func (h *Handler) streamResponses(
	w http.ResponseWriter,
	r *http.Request,
	requestDTO openAIResponsesCreateRequestDTO,
	request domain.StructuredRequest,
) {
	if h.streamer == nil {
		writeJSON(w, http.StatusBadRequest, newOpenAIInvalidRequest("stream", "stream=true is only supported for single_pass backends").Payload)
		return
	}

	stream := newSSEWriter(w)
	if request.RequestID == "" {
		request.RequestID = domain.RequestID(newStreamingRequestID())
	}
	started := false
	sequenceNumber := 1

	start := func(requestID string) error {
		if started {
			return nil
		}
		if err := stream.Start(); err != nil {
			return err
		}
		if err := stream.Data(mapStructuredResponseDomainToResponsesCreatedEventDTO(requestID, requestDTO, h.publicModel)); err != nil {
			return err
		}
		started = true
		return nil
	}

	response, execErr := h.streamer.Stream(r.Context(), request, func(delta domain.CompletionDelta) error {
		if strings.TrimSpace(delta.Content) == "" {
			return nil
		}
		requestID := string(request.RequestID)
		if err := start(requestID); err != nil {
			return err
		}
		event := mapStructuredResponseDomainToResponsesDeltaEventDTO(requestID, sequenceNumber, delta.Content)
		sequenceNumber++
		return stream.Data(event)
	})
	if execErr != nil {
		if !started {
			writeJSON(w, execErr.StatusCode, mapDomainErrorToOpenAIDTO(execErr))
			return
		}
		_ = stream.Data(mapOpenAIErrorToResponsesEventDTO(sequenceNumber, mapDomainErrorToOpenAIDTO(execErr)))
		return
	}

	requestID := string(response.RequestID)
	if err := start(requestID); err != nil {
		return
	}

	outputText := strings.TrimSpace(string(response.Output))
	if err := stream.Data(mapStructuredResponseDomainToResponsesTextDoneEventDTO(requestID, sequenceNumber, outputText)); err != nil {
		return
	}
	sequenceNumber++
	if err := stream.Data(openAIResponsesCompletedEventDTO{
		Type:           "response.completed",
		SequenceNumber: sequenceNumber,
		Response:       mapStructuredResponseDomainToResponsesDTO(response, requestDTO, h.publicModel),
	}); err != nil {
		return
	}
}

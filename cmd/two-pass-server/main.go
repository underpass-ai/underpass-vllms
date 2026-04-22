package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	httpapi "github.com/tgarciai/underpass-vllms/internal/adapters/inbound/httpapi"
	jsonschemaadapter "github.com/tgarciai/underpass-vllms/internal/adapters/outbound/jsonschema"
	openaiadapter "github.com/tgarciai/underpass-vllms/internal/adapters/outbound/openai"
	twopassapp "github.com/tgarciai/underpass-vllms/internal/application/twopass"
	"github.com/tgarciai/underpass-vllms/internal/config"
	domain "github.com/tgarciai/underpass-vllms/internal/domain/twopass"
)

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.LUTC)

	settings, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	formatterProvider, err := openaiadapter.ParseProviderProfile(settings.Pass2.Provider)
	if err != nil {
		logger.Fatalf("parse pass2 provider: %v", err)
	}
	formatter := openaiadapter.NewClient(formatterProvider, settings.Pass2.BaseURL, settings.Pass2.APIKey, settings.Pass2.Timeout, logger)
	validator := jsonschemaadapter.NewValidator()

	appSettings := twopassapp.Settings{
		MaxIntermediateBytes:   settings.MaxIntermediateBytes,
		Pass2RetryCount:        settings.Pass2RetryCount,
		SinglePassSystemPrompt: settings.SinglePassSystemPrompt,
		Versions: twopassapp.Versions{
			Pass1Prompt:      domain.PromptVersion(settings.Pass1PromptVersion),
			Pass2Prompt:      domain.PromptVersion(settings.Pass2PromptVersion),
			SinglePassPrompt: domain.PromptVersion(settings.SinglePassPromptVersion),
			IR:               domain.IRVersion(settings.IRVersion),
		},
		PromptTemplates: twopassapp.PromptTemplates{
			Pass1User:      settings.Pass1UserPromptTemplate,
			Pass2User:      settings.Pass2UserPromptTemplate,
			Pass2RetryHint: settings.Pass2RetryHintTemplate,
			SinglePassUser: settings.SinglePassUserPromptTemplate,
		},
		Pass1:      mapEndpointConfigToPassDefaults(settings.Pass1),
		Pass2:      mapEndpointConfigToPassDefaults(settings.Pass2),
		SinglePass: mapEndpointConfigToPassDefaults(settings.Pass2),
	}

	var executor domain.StructuredExecutionPort
	switch settings.ModelType {
	case config.ModelTypeQwenReasoning:
		reasonerProvider, err := openaiadapter.ParseProviderProfile(settings.Pass1.Provider)
		if err != nil {
			logger.Fatalf("parse pass1 provider: %v", err)
		}
		reasoner := openaiadapter.NewClient(reasonerProvider, settings.Pass1.BaseURL, settings.Pass1.APIKey, settings.Pass1.Timeout, logger)
		executor = twopassapp.NewTwoPassAdapter(appSettings, reasoner, formatter, validator, logger)
	case config.ModelTypeGPTOss, config.ModelTypeGemma4:
		executor = twopassapp.NewSinglePassAdapter(appSettings, formatter, logger)
	default:
		logger.Fatalf("unsupported model type %q", settings.ModelType)
	}

	service := twopassapp.NewService(executor)
	handler := httpapi.NewHandler(service)

	srv := &http.Server{
		Addr:              settings.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Printf("two-pass server listening on %s", settings.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("listen: %v", err)
	}
}

func mapEndpointConfigToPassDefaults(endpoint config.EndpointConfig) twopassapp.PassDefaults {
	return twopassapp.PassDefaults{
		Model:               domain.ModelName(endpoint.Model),
		SystemPrompt:        endpoint.SystemPrompt,
		Temperature:         endpoint.Temperature,
		TopP:                endpoint.TopP,
		TopK:                endpoint.TopK,
		PresencePenalty:     endpoint.PresencePenalty,
		RepetitionPenalty:   endpoint.RepetitionPenalty,
		MaxTokens:           endpoint.MaxTokens,
		ThinkingTokenBudget: endpoint.ThinkingTokenBudget,
		ReasoningEffort:     mapReasoningEffort(endpoint.ReasoningEffort),
		PreserveThinking:    endpoint.PreserveThinking,
	}
}

func mapReasoningEffort(value *string) *domain.ReasoningEffort {
	if value == nil {
		return nil
	}
	mapped := domain.ReasoningEffort(*value)
	return &mapped
}

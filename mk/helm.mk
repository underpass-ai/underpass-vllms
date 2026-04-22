HELM ?= helm
CHART ?= charts/vllm

NAMESPACE ?=
VALUES ?=
EXTRA_ARGS ?=

REASONING_RELEASE ?= underpass-llm-reasoning
STRUCTURED_RELEASE ?= underpass-llm-structured
ORCHESTRATOR_RELEASE ?= underpass-llm-orchestrator

define require_var
	@if [ -z "$($(1))" ]; then \
		echo "$(1) is required"; \
		exit 1; \
	fi
endef

.PHONY: help \
	helm-lint-values \
	helm-template-reasoning \
	helm-upgrade-reasoning \
	helm-uninstall-reasoning \
	helm-template-structured \
	helm-upgrade-structured \
	helm-uninstall-structured \
	helm-template-orchestrator \
	helm-upgrade-orchestrator \
	helm-uninstall-orchestrator

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-30s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

helm-lint-values: ## Lint the chart with an explicit values file: make helm-lint-values VALUES=path/to/file.yaml
	$(call require_var,VALUES)
	$(HELM) lint $(CHART) -f $(VALUES)

helm-template-reasoning: ## Render only the reasoning service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) template $(REASONING_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		-f $(VALUES) \
		--set reasoning.enabled=true \
		--set structured.enabled=false \
		--set orchestrator.enabled=false \
		$(EXTRA_ARGS)

helm-upgrade-reasoning: ## Install or upgrade only the reasoning service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) upgrade --install $(REASONING_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		--create-namespace \
		-f $(VALUES) \
		--set reasoning.enabled=true \
		--set structured.enabled=false \
		--set orchestrator.enabled=false \
		$(EXTRA_ARGS)

helm-uninstall-reasoning: ## Uninstall the reasoning release
	$(call require_var,NAMESPACE)
	$(HELM) uninstall $(REASONING_RELEASE) -n $(NAMESPACE)

helm-template-structured: ## Render only the structured service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) template $(STRUCTURED_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		-f $(VALUES) \
		--set reasoning.enabled=false \
		--set structured.enabled=true \
		--set orchestrator.enabled=false \
		$(EXTRA_ARGS)

helm-upgrade-structured: ## Install or upgrade only the structured service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) upgrade --install $(STRUCTURED_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		--create-namespace \
		-f $(VALUES) \
		--set reasoning.enabled=false \
		--set structured.enabled=true \
		--set orchestrator.enabled=false \
		$(EXTRA_ARGS)

helm-uninstall-structured: ## Uninstall the structured release
	$(call require_var,NAMESPACE)
	$(HELM) uninstall $(STRUCTURED_RELEASE) -n $(NAMESPACE)

helm-template-orchestrator: ## Render only the orchestrator service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) template $(ORCHESTRATOR_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		-f $(VALUES) \
		--set reasoning.enabled=false \
		--set structured.enabled=false \
		--set orchestrator.enabled=true \
		$(EXTRA_ARGS)

helm-upgrade-orchestrator: ## Install or upgrade only the orchestrator service
	$(call require_var,NAMESPACE)
	$(call require_var,VALUES)
	$(HELM) upgrade --install $(ORCHESTRATOR_RELEASE) $(CHART) \
		-n $(NAMESPACE) \
		--create-namespace \
		-f $(VALUES) \
		--set reasoning.enabled=false \
		--set structured.enabled=false \
		--set orchestrator.enabled=true \
		$(EXTRA_ARGS)

helm-uninstall-orchestrator: ## Uninstall the orchestrator release
	$(call require_var,NAMESPACE)
	$(HELM) uninstall $(ORCHESTRATOR_RELEASE) -n $(NAMESPACE)

{{- define "underpass-llm.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "underpass-llm.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "underpass-llm.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "underpass-llm.labels" -}}
helm.sh/chart: {{ include "underpass-llm.chart" . }}
app.kubernetes.io/name: {{ include "underpass-llm.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "underpass-llm.componentName" -}}
{{- printf "%s-%s" (include "underpass-llm.fullname" .root) .component | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "underpass-llm.componentLabels" -}}
{{ include "underpass-llm.labels" .root }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{- define "underpass-llm.componentSelectorLabels" -}}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{- define "underpass-llm.cacheName" -}}
{{- printf "%s-hf-cache" (include "underpass-llm.fullname" .) -}}
{{- end -}}

{{- define "underpass-llm.orchestratorPass1BaseURL" -}}
{{- required "orchestrator.pass1.baseURL is required when orchestrator.enabled=true" .Values.orchestrator.pass1.baseURL -}}
{{- end -}}

{{- define "underpass-llm.orchestratorPass1Model" -}}
{{- required "orchestrator.pass1.model is required when orchestrator.enabled=true" .Values.orchestrator.pass1.model -}}
{{- end -}}

{{- define "underpass-llm.orchestratorPass2BaseURL" -}}
{{- required "orchestrator.pass2.baseURL is required when orchestrator.enabled=true" .Values.orchestrator.pass2.baseURL -}}
{{- end -}}

{{- define "underpass-llm.orchestratorPass2Model" -}}
{{- required "orchestrator.pass2.model is required when orchestrator.enabled=true" .Values.orchestrator.pass2.model -}}
{{- end -}}

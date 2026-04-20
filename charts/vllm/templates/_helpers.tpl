{{- define "vllm.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "vllm.labels" -}}
helm.sh/chart: vllm-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "vllm.cacheName" -}}
{{- printf "%s-hf-cache" (include "vllm.fullname" .) -}}
{{- end -}}

{{/*
Create a default fully qualified app name, to use as base bame for all ressources.
Use the release name by default
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "kindex.baseName" -}}
{{- if .Values.baseNameOverride }}
{{- .Values.baseNameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kindex.chartName" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Common labels
*/}}
{{- define "kindex.labels" -}}
helm.sh/chart: {{ include "kindex.chartName" . }}
{{ include "kindex.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{/*
Controller Selector labels
*/}}
{{- define "kindex.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kindex.baseName" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the deployment to use
*/}}
{{- define "kindex.deploymentName" -}}
{{- default (printf "%s" (include "kindex.baseName" .)) .Values.deploymentName }}
{{- end }}



{{/*
Create the name of the tls certificate
*/}}
{{- define "kindex.certificateName" -}}
{{- default (printf "%s" (include "kindex.baseName" .)) .Values.certificateName }}
{{- end }}


{{/*
Create the name of the secret hosting the server certificate
*/}}
{{- define "kindex.certificateSecretName" -}}
{{- default (printf "%s-cert" (include "kindex.baseName" .)) .Values.certificateSecretName }}
{{- end }}


{{/*
Create the name of the service
*/}}
{{- define "kindex.serviceName" -}}
{{- default (printf "%s" (include "kindex.baseName" .)) .Values.serviceName }}
{{- end }}


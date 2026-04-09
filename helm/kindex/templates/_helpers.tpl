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

{{/*
Create the name of the ingrss
*/}}
{{- define "kindex.ingressName" -}}
{{- default (printf "%s" (include "kindex.baseName" .)) .Values.ingressName }}
{{- end }}

{{/*
Create the name of the ingrss
*/}}
{{- define "kindex.networkPolicyName" -}}
{{- default (printf "allow-%s" (include "kindex.baseName" .)) .Values.networkPolicyName }}
{{- end }}

{{/*
Service account name (created when serviceAccount.create, or explicit name for an existing SA).
*/}}
{{- define "kindex.serviceAccountName" -}}
{{- if .Values.serviceAccountName }}
{{- .Values.serviceAccountName }}
{{- else }}
{{- include "kindex.baseName" . }}
{{- end }}
{{- end }}

{{/*
ClusterRole for listing Ingresses across all namespaces (cluster-scoped name).
*/}}
{{- define "kindex.clusterRoleName" -}}
{{- if .Values.clusterRoleName }}
{{- .Values.clusterRoleName }}
{{- else }}
{{- printf "%s-%s-kindex-ingress-reader" .Release.Namespace (include "kindex.baseName" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
ClusterRoleBinding tying the ServiceAccount to the ClusterRole.
*/}}
{{- define "kindex.clusterRoleBindingName" -}}
{{- if .Values.clusterRoleBindingName }}
{{- .Values.clusterRoleBindingName }}
{{- else }}
{{- printf "%s-%s-kindex-ingress-crb" .Release.Namespace (include "kindex.baseName" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}


{{/*
Expand the name of the chart.
*/}}
{{- define "wealth-warden.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "wealth-warden.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "wealth-warden.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "wealth-warden.labels" -}}
helm.sh/chart: {{ include "wealth-warden.chart" . }}
{{ include "wealth-warden.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "wealth-warden.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wealth-warden.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "wealth-warden.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "wealth-warden.fullname" .) .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the config secret
*/}}
{{- define "wealth-warden.secretName" -}}
{{- printf "%s-secret" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
Create the name of the config configmap
*/}}
{{- define "wealth-warden.configMapName" -}}
{{- printf "%s-config" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
PostgreSQL host
*/}}
{{- define "wealth-warden.postgresHost" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "%s-postgresql-hl.%s.svc.cluster.local" .Release.Name .Release.Namespace }}
{{- else }}
{{- .Values.config.postgres.host | default "postgres" }}
{{- end }}
{{- end }}

{{/*
PostgreSQL user
*/}}
{{- define "wealth-warden.postgresUser" -}}
{{- if .Values.postgresql.enabled }}
{{- .Values.postgresql.auth.username | default "postgres" }}
{{- else }}
{{- .Values.config.postgres.user }}
{{- end }}
{{- end }}

{{/*
PostgreSQL database
*/}}
{{- define "wealth-warden.postgresDatabase" -}}
{{- if .Values.postgresql.enabled }}
{{- .Values.postgresql.auth.database | default "wealth_warden" }}
{{- else }}
{{- .Values.config.postgres.db }}
{{- end }}
{{- end }}

{{/*
API component name
*/}}
{{- define "wealth-warden.api.name" -}}
{{- printf "%s-api" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
API selector labels
*/}}
{{- define "wealth-warden.api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wealth-warden.name" . }}-api
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: api
{{- end }}

{{/*
WebUI component name
*/}}
{{- define "wealth-warden.webui.name" -}}
{{- printf "%s-webui" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
WebUI selector labels
*/}}
{{- define "wealth-warden.webui.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wealth-warden.name" . }}-webui
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: webui
{{- end }}

{{/*
API service name
*/}}
{{- define "wealth-warden.api.serviceName" -}}
{{- printf "%s-api" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
WebUI service name
*/}}
{{- define "wealth-warden.webui.serviceName" -}}
{{- printf "%s-webui" (include "wealth-warden.fullname" .) }}
{{- end }}

{{/*
API ConfigMap checksum - checksums the ConfigMap values
*/}}
{{- define "wealth-warden.api.configChecksum" -}}
{{- $configValues := dict -}}
{{- $configValues = set $configValues "release" .Values.config.release -}}
{{- $configValues = set $configValues "financeApiBaseUrl" .Values.config.financeApiBaseUrl -}}
{{- $configValues = set $configValues "httpServer" .Values.config.httpServer -}}
{{- $configValues = set $configValues "webClient" .Values.config.webClient -}}
{{- $configValues = set $configValues "postgresHost" (include "wealth-warden.postgresHost" .) -}}
{{- $configValues = set $configValues "postgresUser" (include "wealth-warden.postgresUser" .) -}}
{{- $configValues = set $configValues "postgresPort" .Values.config.postgres.port -}}
{{- $configValues = set $configValues "postgresDb" (include "wealth-warden.postgresDatabase" .) -}}
{{- $configValues = set $configValues "cors" .Values.config.cors -}}
{{- $configValues = set $configValues "mailer" .Values.config.mailer -}}
{{- $configValues = set $configValues "seed" .Values.config.seed -}}
{{- $configValues | toJson | sha256sum -}}
{{- end -}}

{{/*
WebUI ConfigMap checksum - checksums the nginx config values
*/}}
{{- define "wealth-warden.webui.configChecksum" -}}
{{- $nginxValues := dict -}}
{{- $nginxValues = set $nginxValues "apiServiceName" (include "wealth-warden.api.serviceName" .) -}}
{{- $nginxValues = set $nginxValues "apiServicePort" .Values.api.service.port -}}
{{- $nginxValues | toJson | sha256sum -}}
{{- end -}}

{{/*
Secret checksum - checksums secret values if they exist
*/}}
{{- define "wealth-warden.secretChecksum" -}}
{{- if .Values.secrets.create -}}
{{- $secretData := dict -}}
{{- if .Values.secrets.postgresPassword }}{{- $secretData = set $secretData "postgres-password" .Values.secrets.postgresPassword -}}{{- end -}}
{{- if .Values.secrets.jwtWebClientAccess }}{{- $secretData = set $secretData "jwt-web-client-access" .Values.secrets.jwtWebClientAccess -}}{{- end -}}
{{- if .Values.secrets.jwtWebClientRefresh }}{{- $secretData = set $secretData "jwt-web-client-refresh" .Values.secrets.jwtWebClientRefresh -}}{{- end -}}
{{- if .Values.secrets.jwtWebClientEncodeId }}{{- $secretData = set $secretData "jwt-web-client-encode-id" .Values.secrets.jwtWebClientEncodeId -}}{{- end -}}
{{- if .Values.secrets.mailerPassword }}{{- $secretData = set $secretData "mailer-password" .Values.secrets.mailerPassword -}}{{- end -}}
{{- if .Values.secrets.superAdminPassword }}{{- $secretData = set $secretData "super-admin-password" .Values.secrets.superAdminPassword -}}{{- end -}}
{{- $secretData | toJson | sha256sum -}}
{{- end -}}
{{- end -}}


{{/*
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
*/}}
{{- define "github-deployment-bridge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "github-deployment-bridge.fullname" -}}
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

{{- define "github-deployment-bridge.labels" -}}
helm.sh/chart: {{ include "github-deployment-bridge.chart" . }}
{{ include "github-deployment-bridge.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "github-deployment-bridge.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "github-deployment-bridge.selectorLabels" -}}
app.kubernetes.io/name: {{ include "github-deployment-bridge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "github-deployment-bridge.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "github-deployment-bridge.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "github-deployment-bridge.secretName" -}}
{{- if .Values.github.existingSecret }}
{{- .Values.github.existingSecret }}
{{- else }}
{{- include "github-deployment-bridge.fullname" . }}-github
{{- end }}
{{- end }}

{{/*
Payload hashed into checksum/config so env / listen-port / registry changes
roll the Deployment. Keep selectorLabels out of this — they must stay stable.
*/}}
{{- define "github-deployment-bridge.configChecksum" -}}
{{- toYaml (dict
  "config" .Values.config
  "containerPorts" .Values.containerPorts
  "registry" .Values.registry
  "probes" .Values.probes
  "existingSecret" .Values.github.existingSecret
) -}}
{{- end }}

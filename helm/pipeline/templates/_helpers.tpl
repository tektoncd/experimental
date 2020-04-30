{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "pipeline.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pipeline.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "pipeline.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "pipeline.serviceAccountName" -}}
{{- if .Values.rbac.create -}}
{{- template "pipeline.fullname" . -}}
{{- else -}}
{{- required "A service account name is required" .Values.rbac.serviceAccountName -}}
{{- end -}}
{{- end -}}

{{/*
Create base labels
*/}}
{{- define "pipeline.baseLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: tekton-pipelines
{{- end -}}

{{/*
Create helm labels
*/}}
{{- define "pipeline.helmLabels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ template "pipeline.chart" . }}
{{- end -}}

{{/*
Create version labels
*/}}
{{- define "pipeline.versionLabels" -}}
app.kubernetes.io/version: {{ .Values.version | quote }}
pipeline.tekton.dev/release: {{ .Values.version | quote }}
version: {{ .Values.version | quote }}
{{- end -}}

{{/*
Create component labels
*/}}
{{- define "pipeline.componentLabels" -}}
app.kubernetes.io/component: {{ . }}
{{- end -}}

{{/*
Create name labels
*/}}
{{- define "pipeline.nameLabels" -}}
app.kubernetes.io/name: {{ . }}
{{- end -}}

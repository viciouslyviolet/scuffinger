{{/*
Expand the name of the chart.
*/}}
{{- define "scuffinger.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this.
*/}}
{{- define "scuffinger.fullname" -}}
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
{{- define "scuffinger.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels — applied to every resource.
Compatible with ArgoCD label-based tracking.
*/}}
{{- define "scuffinger.labels" -}}
helm.sh/chart: {{ include "scuffinger.chart" . }}
{{ include "scuffinger.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: scuffinger
{{- with .Values.global.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "scuffinger.selectorLabels" -}}
app.kubernetes.io/name: {{ include "scuffinger.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Component labels helper — call with (list $ "component-name").
*/}}
{{- define "scuffinger.componentLabels" -}}
{{- $root := index . 0 -}}
{{- $component := index . 1 -}}
{{ include "scuffinger.labels" $root }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Component selector labels.
*/}}
{{- define "scuffinger.componentSelectorLabels" -}}
{{- $root := index . 0 -}}
{{- $component := index . 1 -}}
{{ include "scuffinger.selectorLabels" $root }}
app.kubernetes.io/component: {{ $component }}
{{- end }}

{{/*
Service account name for the app.
*/}}
{{- define "scuffinger.serviceAccountName" -}}
{{- if .Values.app.serviceAccount.create }}
{{- default (include "scuffinger.fullname" .) .Values.app.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.app.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Common annotations — applied to every resource.
Includes ArgoCD sync-wave support.
*/}}
{{- define "scuffinger.annotations" -}}
{{- with .Values.global.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Image helper: builds the full image string.
Usage: {{ include "scuffinger.image" (dict "image" .Values.app.image "chart" .Chart) }}
*/}}
{{- define "scuffinger.image" -}}
{{- $tag := .image.tag | default .chart.AppVersion -}}
{{- printf "%s:%s" .image.repository $tag -}}
{{- end }}

{{/*
Collector image helper — falls back to the app image when collector fields are empty.
Usage: {{ include "scuffinger.collectorImage" . }}
*/}}
{{- define "scuffinger.collectorImage" -}}
{{- $repo := .Values.collector.image.repository | default .Values.app.image.repository -}}
{{- $tag  := .Values.collector.image.tag | default (.Values.app.image.tag | default .Chart.AppVersion) -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end }}

{{/*
Collector image pull policy — falls back to the app pull policy.
*/}}
{{- define "scuffinger.collectorImagePullPolicy" -}}
{{- .Values.collector.image.pullPolicy | default .Values.app.image.pullPolicy -}}
{{- end }}

{{/*
Image pull secrets — merges global and component-level.
*/}}
{{- define "scuffinger.imagePullSecrets" -}}
{{- $secrets := concat (.Values.global.imagePullSecrets | default list) (.secrets | default list) -}}
{{- if $secrets }}
imagePullSecrets:
{{- range $secrets }}
  - name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
PostgreSQL host — uses internal service name when postgresql.enabled, otherwise falls back.
*/}}
{{- define "scuffinger.postgresql.host" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "%s-postgresql" (include "scuffinger.fullname" .) }}
{{- else }}
{{- required "postgresql.externalHost is required when postgresql.enabled=false" .Values.postgresql.externalHost }}
{{- end }}
{{- end }}

{{/*
ValKey host.
*/}}
{{- define "scuffinger.valkey.host" -}}
{{- if .Values.valkey.enabled }}
{{- printf "%s-valkey" (include "scuffinger.fullname" .) }}
{{- else }}
{{- required "valkey.externalHost is required when valkey.enabled=false" .Values.valkey.externalHost }}
{{- end }}
{{- end }}

{{/*
Prometheus host.
*/}}
{{- define "scuffinger.prometheus.host" -}}
{{- printf "%s-prometheus" (include "scuffinger.fullname" .) }}
{{- end }}

{{/*
Loki host.
*/}}
{{- define "scuffinger.loki.host" -}}
{{- printf "%s-loki" (include "scuffinger.fullname" .) }}
{{- end }}


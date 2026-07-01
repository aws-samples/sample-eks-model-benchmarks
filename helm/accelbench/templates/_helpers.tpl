{{/*
Expand the name of the chart.
*/}}
{{- define "accelbench.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "accelbench.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- printf "%s" $name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "accelbench.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
API selector labels.
*/}}
{{- define "accelbench.api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "accelbench.name" . }}-api
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Web selector labels.
*/}}
{{- define "accelbench.web.selectorLabels" -}}
app.kubernetes.io/name: {{ include "accelbench.name" . }}-web
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
API service account name.
*/}}
{{- define "accelbench.api.serviceAccountName" -}}
{{- if .Values.api.serviceAccount.create }}
{{- default (printf "%s-api" (include "accelbench.fullname" .)) .Values.api.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.api.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database URL — either from values or from existing secret.
If existingSecret is set, the env var references the secret.
Otherwise it's constructed from components or used directly from values.database.url.
*/}}
{{- define "accelbench.databaseURL" -}}
{{- if .Values.database.url }}
{{- .Values.database.url }}
{{- else }}
{{- printf "postgres://%s@%s:%d/%s?sslmode=%s" .Values.database.username .Values.database.host (int .Values.database.port) .Values.database.name .Values.database.sslmode }}
{{- end }}
{{- end }}

{{/*
HuggingFace secret name.
*/}}
{{- define "accelbench.hfSecretName" -}}
{{- if .Values.huggingface.existingSecret }}
{{- .Values.huggingface.existingSecret }}
{{- else }}
{{- printf "%s-hf-token" (include "accelbench.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Database secret name.
*/}}
{{- define "accelbench.dbSecretName" -}}
{{- if .Values.database.existingSecret }}
{{- .Values.database.existingSecret }}
{{- else }}
{{- printf "%s-db" (include "accelbench.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Resolve the full image reference for a service.
Usage: {{ include "accelbench.image" (dict "service" "api" "Values" .Values "Chart" .Chart) }}

Priority:
  1. image.<service>.repository (if set) — full override
  2. image.registry / accelbench-<service> — default from public ECR

Tag priority:
  1. image.<service>.tag (if set)
  2. image.tag (global override)
  3. Chart.AppVersion
*/}}
{{- define "accelbench.image" -}}
{{- $svc := index .Values.image .service -}}
{{- $repo := $svc.repository -}}
{{- if not $repo -}}
  {{- $repo = printf "%s/accelbench-%s" .Values.image.registry .service -}}
{{- end -}}
{{- $tag := $svc.tag -}}
{{- if not $tag -}}
  {{- $tag = .Values.image.tag -}}
{{- end -}}
{{- if not $tag -}}
  {{- $tag = .Chart.AppVersion -}}
{{- end -}}
{{- printf "%s:%s" $repo $tag -}}
{{- end }}

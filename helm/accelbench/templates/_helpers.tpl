{{/*
Expand the name of the chart.
*/}}
{{- define "accelbench.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
Follows the standard Helm pattern: release-name aware, but collapses to just
the chart name when the release name already contains it (e.g. the documented
`helm install accelbench ...`), so rendered object names stay `accelbench-*`.
*/}}
{{- define "accelbench.fullname" -}}
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
Namespace to deploy into. Defaults to the release namespace (idiomatic Helm),
but honours .Values.namespace for backward compatibility with existing
Terraform / README flows that pin `accelbench`.
*/}}
{{- define "accelbench.namespace" -}}
{{- default .Release.Namespace .Values.namespace }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "accelbench.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "accelbench.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: accelbench
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
If existingSecret is set, the env var references the secret directly (this is
the recommended path: Terraform builds a pre-encoded DATABASE_URL there).

Otherwise the DSN is resolved here:
  1. .Values.database.url        — used VERBATIM. You are responsible for
                                    percent-encoding the password yourself
                                    (RDS-managed passwords contain characters
                                    like [ # | > that break URL parsers —
                                    "invalid IP-literal" — if left raw).
  2. component fields            — assembled from host/port/name/username and
                                    .Values.database.password. The password is
                                    percent-encoded here via urlquery so special
                                    characters are safe automatically.
*/}}
{{- define "accelbench.databaseURL" -}}
{{- if .Values.database.url }}
{{- .Values.database.url }}
{{- else -}}
{{- $user := .Values.database.username -}}
{{- $pass := .Values.database.password | default "" -}}
{{- $userinfo := $user -}}
{{- if $pass -}}
{{- $userinfo = printf "%s:%s" $user (urlquery $pass) -}}
{{- end -}}
{{- printf "postgres://%s@%s:%d/%s?sslmode=%s" $userinfo .Values.database.host (int .Values.database.port) .Values.database.name .Values.database.sslmode -}}
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

{{/*
Pod-level security context (shared default).
*/}}
{{- define "accelbench.podSecurityContext" -}}
{{- toYaml .Values.podSecurityContext }}
{{- end }}

{{/*
Container-level security context (shared default).
*/}}
{{- define "accelbench.containerSecurityContext" -}}
{{- toYaml .Values.containerSecurityContext }}
{{- end }}

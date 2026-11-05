{{- define "squirrelbot.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "squirrelbot.fullname" -}}
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

{{- define "squirrelbot.secretName" -}}
{{- if .Values.existingSecret }}
{{- .Values.existingSecret }}
{{- else }}
{{- include "squirrelbot.fullname" . }}
{{- end }}
{{- end }}

{{- define "squirrelbot.host" -}}
{{- $parsed := urlParse (required "publicUrl is required" .Values.publicUrl) -}}
{{- required "publicUrl must include a host" $parsed.hostname -}}
{{- end }}

{{- define "squirrelbot.image" -}}
{{- $image := .Values.image | default dict }}
{{- $repository := $image.repository | default "ghcr.io/boxingoctopuscreative/squirrelbot" }}
{{- $tag := $image.tag | default "latest" }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}

{{- define "squirrelbot.sslMode" -}}
{{- $postgres := .Values.postgres | default dict -}}
{{- $mode := $postgres.sslMode -}}
{{- if not $mode -}}
{{- if $postgres.bundled -}}
{{- $mode = "disable" -}}
{{- else -}}
{{- $mode = "require" -}}
{{- end -}}
{{- end -}}
{{- $allowed := list "disable" "allow" "prefer" "require" "verify-ca" "verify-full" -}}
{{- if not (has $mode $allowed) -}}
{{- fail "postgres.sslMode must be disable, allow, prefer, require, verify-ca, or verify-full" -}}
{{- end -}}
{{- $mode -}}
{{- end }}

{{- define "squirrelbot.databaseUrl" -}}
{{- $mode := include "squirrelbot.sslMode" . -}}
{{- if .Values.postgres.bundled -}}
{{- printf "postgres://%s:%s@%s-postgres:5432/%s?sslmode=%s" .Values.postgres.auth.username (.Values.postgres.auth.password | urlquery) (include "squirrelbot.fullname" .) .Values.postgres.auth.database $mode -}}
{{- else -}}
{{- $raw := required "secrets.databaseUrl is required when postgres.bundled is false" .Values.secrets.databaseUrl -}}
{{- if contains "sslmode=" $raw -}}
{{- $raw -}}
{{- else if contains "?" $raw -}}
{{- printf "%s&sslmode=%s" $raw $mode -}}
{{- else -}}
{{- printf "%s?sslmode=%s" $raw $mode -}}
{{- end -}}
{{- end -}}
{{- end }}

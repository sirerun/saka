{{- define "saka.namespace" -}}
{{ .Values.namespace.name | default .Release.Namespace }}
{{- end -}}

{{/* Determine osm namespace */}}
{{- define "osm.namespace" -}} 
{{ default .Release.Namespace .Values.OpenServiceMesh.osmNamespace}} 
{{- end -}}

{{/* Default tracing address */}}
{{- define "osm.tracingAddress" -}}
{{- $address := printf "jaeger.%s.svc.cluster.local" (include "osm.namespace" .) -}}
{{ default $address .Values.OpenServiceMesh.tracing.address}} 
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "osm.labels" -}}
app.kubernetes.io/name: openservicemesh.io
app.kubernetes.io/instance: {{ .Values.OpenServiceMesh.meshName }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end -}}

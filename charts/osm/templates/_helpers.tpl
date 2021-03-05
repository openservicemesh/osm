{{/* Determine osm namespace */}}
{{- define "osm.namespace" -}} 
{{ default .Release.Namespace .Values.OpenServiceMesh.osmNamespace}} 
{{- end -}}

{{/* Default tracing address */}}
{{- define "osm.tracingAddress" -}}
{{- $address := printf "jaeger.%s.svc.cluster.local" (include "osm.namespace" .) -}}
{{ default $address .Values.OpenServiceMesh.tracing.address}} 
{{- end -}}

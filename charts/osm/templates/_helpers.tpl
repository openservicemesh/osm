{{/* Determine osm namespace */}}
{{- define "osm.namespace" -}} 
{{ default .Release.Namespace .Values.OpenServiceMesh.osmNamespace}} 
{{- end -}}

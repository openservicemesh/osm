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

{{/* Security context values that ensure restricted access to host resources */}}
{{- define "restricted.securityContext" -}}
securityContext:
    runAsUser: 1000
    runAsGroup: 3000
    fsGroup: 2000
    supplementalGroups: [5555]
{{- end -}}

{{/* Security context values for fluentbit */}}
{{- define "fluentbit.securityContext" -}}
securityContext:
    runAsUser: 0
    capabilities:
        drop:
            - ALL
{{- end -}}

{{/* Resource validator webhook name */}}
{{- define "osm.validatorWebhookConfigName" -}}
{{- $validatorWebhookConfigName := printf "osm-validator-mesh-%s" .Values.OpenServiceMesh.meshName -}}
{{ default $validatorWebhookConfigName .Values.OpenServiceMesh.validatorWebhook.webhookConfigurationName}}
{{- end -}}

{{/* osm-controller image */}}
{{- define "osmController.image" -}}
{{- if .Values.OpenServiceMesh.image.tag -}}
{{- printf "%s/osm-controller:%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.tag -}}
{{- else -}}
{{- printf "%s/osm-controller@%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.digest.osmController -}}
{{- end -}}
{{- end -}}

{{/* osm-injector image */}}
{{- define "osmInjector.image" -}}
{{- if .Values.OpenServiceMesh.image.tag -}}
{{- printf "%s/osm-injector:%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.tag -}}
{{- else -}}
{{- printf "%s/osm-injector@%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.digest.osmInjector -}}
{{- end -}}
{{- end -}}

{{/* Sidecar init image */}}
{{- define "osmSidecarInit.image" -}}
{{- if .Values.OpenServiceMesh.image.tag -}}
{{- printf "%s/init:%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.tag -}}
{{- else -}}
{{- printf "%s/init@%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.digest.osmSidecarInit -}}
{{- end -}}
{{- end -}}

{{/* osm-bootstrap image */}}
{{- define "osmBootstrap.image" -}}
{{- if .Values.OpenServiceMesh.image.tag -}}
{{- printf "%s/osm-bootstrap:%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.tag -}}
{{- else -}}
{{- printf "%s/osm-bootstrap@%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.digest.osmBootstrap -}}
{{- end -}}
{{- end -}}

{{/* osm-crds image */}}
{{- define "osmCRDs.image" -}}
{{- if .Values.OpenServiceMesh.image.tag -}}
{{- printf "%s/osm-crds:%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.tag -}}
{{- else -}}
{{- printf "%s/osm-crds@%s" .Values.OpenServiceMesh.image.registry .Values.OpenServiceMesh.image.digest.osmCRDs -}}
{{- end -}}
{{- end -}}
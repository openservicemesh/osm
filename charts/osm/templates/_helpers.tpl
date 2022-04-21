{{/* Determine osm namespace */}}
{{- define "osm.namespace" -}}
{{ default .Release.Namespace .Values.osm.osmNamespace}}
{{- end -}}

{{/* Default tracing address */}}
{{- define "osm.tracingAddress" -}}
{{- $address := printf "jaeger.%s.svc.cluster.local" (include "osm.namespace" .) -}}
{{ default $address .Values.osm.tracing.address}}
{{- end -}}

{{/* Labels to be added to all resources */}}
{{- define "osm.labels" -}}
app.kubernetes.io/name: openservicemesh.io
app.kubernetes.io/instance: {{ .Values.osm.meshName }}
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
{{- $validatorWebhookConfigName := printf "osm-validator-mesh-%s" .Values.osm.meshName -}}
{{ default $validatorWebhookConfigName .Values.osm.validatorWebhook.webhookConfigurationName}}
{{- end -}}

{{/* osm-controller image */}}
{{- define "osmController.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmController .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmController .Values.osm.image.digest.osmController -}}
{{- end -}}
{{- end -}}

{{/* osm-injector image */}}
{{- define "osmInjector.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmInjector .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmInjector .Values.osm.image.digest.osmInjector -}}
{{- end -}}
{{- end -}}

{{/* Sidecar init image */}}
{{- define "osmSidecarInit.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmSidecarInit .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmSidecarInit .Values.osm.image.digest.osmSidecarInit -}}
{{- end -}}
{{- end -}}

{{/* osm-bootstrap image */}}
{{- define "osmBootstrap.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmBootstrap .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmBootstrap .Values.osm.image.digest.osmBootstrap -}}
{{- end -}}
{{- end -}}

{{/* osm-crds image */}}
{{- define "osmCRDs.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmCRDs .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmCRDs .Values.osm.image.digest.osmCRDs -}}
{{- end -}}
{{- end -}}

{{/* osm-preinstall image */}}
{{- define "osmPreinstall.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmPreinstall .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmPreinstall .Values.osm.image.digest.osmPreinstall -}}
{{- end -}}
{{- end -}}

{{/* osm-healthcheck image */}}
{{- define "osmHealthcheck.image" -}}
{{- if .Values.osm.image.tag -}}
{{- printf "%s/%s:%s" .Values.osm.image.registry .Values.osm.image.name.osmHealthcheck .Values.osm.image.tag -}}
{{- else -}}
{{- printf "%s/%s@%s" .Values.osm.image.registry .Values.osm.image.name.osmHealthcheck .Values.osm.image.digest.osmHealthcheck -}}
{{- end -}}
{{- end -}}

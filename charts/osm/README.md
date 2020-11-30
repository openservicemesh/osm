# osm

![Version: 0.4.2](https://img.shields.io/badge/Version-0.4.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.4.2](https://img.shields.io/badge/AppVersion-v0.4.2-informational?style=flat-square)

A Helm chart to install the OSM control plane on Kubernetes

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| OpenServiceMesh.caBundleSecretName | string | `"osm-ca-bundle"` |  |
| OpenServiceMesh.certificateManager | string | `"tresor"` |  |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager"` |  |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` |  |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` |  |
| OpenServiceMesh.deployJaeger | bool | `true` |  |
| OpenServiceMesh.enableBackpressureExperimental | bool | `false` |  |
| OpenServiceMesh.enableDebugServer | bool | `false` |  |
| OpenServiceMesh.enableEgress | bool | `false` |  |
| OpenServiceMesh.deployGrafana | bool | `false` |  |
| OpenServiceMesh.enableFluentbit | bool | `false` |  |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` |  |
| OpenServiceMesh.deployPrometheus | bool | `false` |  |
| OpenServiceMesh.enablePrometheusScraping | bool | `true` | Cannot be false if `deployPrometheus` is true |
| OpenServiceMesh.envoyLogLevel | string | `"error"` |  |
| OpenServiceMesh.fluentBit.name | string | `"fluentbit-logger"` |  |
| OpenServiceMesh.fluentBit.registry | string | `"fluent"` |  |
| OpenServiceMesh.fluentBit.tag | string | `"1.6.4"` |  |
| OpenServiceMesh.fluentBit.pullPolicy | string | `"IfNotPresent"` |  |
| OpenServiceMesh.fluentBit.defaultOutput | string | `"stdout"` |  |
| OpenServiceMesh.fluentBit.allowCustomOutput | bool | `"true"` |  |
| OpenServiceMesh.grafana.port | int | `3000` |  |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` |  |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` |  |
| OpenServiceMesh.image.tag | string | `"v0.4.2"` |  |
| OpenServiceMesh.imagePullSecrets | list | `[]` |  |
| OpenServiceMesh.meshName | string | `"osm"` |  |
| OpenServiceMesh.prometheus.port | int | `7070` |  |
| OpenServiceMesh.prometheus.retention.time | string | `"15d"` |  |
| OpenServiceMesh.replicaCount | int | `1` |  |
| OpenServiceMesh.serviceCertValidityDuration | string | `"24h"` |  |
| OpenServiceMesh.sidecarImage | string | `"envoyproxy/envoy-alpine:v1.15.0"` |  |
| OpenServiceMesh.tracing.address | string | `"jaeger.osm-system.svc.cluster.local"` |  |
| OpenServiceMesh.tracing.enable | bool | `true` |  |
| OpenServiceMesh.tracing.endpoint | string | `"/api/v2/spans"` |  |
| OpenServiceMesh.tracing.port | int | `9411` |  |
| OpenServiceMesh.useHTTPSIngress | bool | `false` |  |
| OpenServiceMesh.vault.host | string | `nil` |  |
| OpenServiceMesh.vault.protocol | string | `"http"` |  |
| OpenServiceMesh.vault.role | string | `"openservicemesh"` |  |
| OpenServiceMesh.vault.token | string | `nil` |  |
| OpenServiceMesh.enforceSingleMesh | bool | `"false"` |  |

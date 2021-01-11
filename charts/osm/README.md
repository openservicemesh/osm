# osm

![Version: 0.6.1](https://img.shields.io/badge/Version-0.6.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.6.1](https://img.shields.io/badge/AppVersion-v0.6.1-informational?style=flat-square)

A Helm chart to install the OSM control plane on Kubernetes

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| OpenServiceMesh.caBundleSecretName | string | `"osm-ca-bundle"` |  |
| OpenServiceMesh.certificateManager | string | `"tresor"` |  |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager"` |  |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` |  |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` |  |
| OpenServiceMesh.connectVault | bool | `true` |  |
| OpenServiceMesh.controllerLogLevel | string | `"trace"` |  |
| OpenServiceMesh.deployGrafana | bool | `false` |  |
| OpenServiceMesh.deployJaeger | bool | `true` |  |
| OpenServiceMesh.deployPrometheus | bool | `false` |  |
| OpenServiceMesh.enableBackpressureExperimental | bool | `false` |  |
| OpenServiceMesh.enableDebugServer | bool | `false` |  |
| OpenServiceMesh.enableEgress | bool | `false` |  |
| OpenServiceMesh.enableFluentbit | bool | `false` |  |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` |  |
| OpenServiceMesh.enablePrometheusScraping | bool | `true` |  |
| OpenServiceMesh.enforceSingleMesh | bool | `false` |  |
| OpenServiceMesh.envoyLogLevel | string | `"error"` |  |
| OpenServiceMesh.fluentBit.enableProxySupport | bool | `false` |  |
| OpenServiceMesh.fluentBit.httpProxy | string | `""` |  |
| OpenServiceMesh.fluentBit.httpsProxy | string | `""` |  |
| OpenServiceMesh.fluentBit.logLevel | string | `"error"` |  |
| OpenServiceMesh.fluentBit.name | string | `"fluentbit-logger"` |  |
| OpenServiceMesh.fluentBit.outputPlugin | string | `"stdout"` |  |
| OpenServiceMesh.fluentBit.primaryKey | string | `""` |  |
| OpenServiceMesh.fluentBit.pullPolicy | string | `"IfNotPresent"` |  |
| OpenServiceMesh.fluentBit.registry | string | `"fluent"` |  |
| OpenServiceMesh.fluentBit.tag | string | `"1.6.4"` |  |
| OpenServiceMesh.fluentBit.workspaceId | string | `""` |  |
| OpenServiceMesh.grafana.enableRemoteRendering | bool | `false` |  |
| OpenServiceMesh.grafana.port | int | `3000` |  |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` |  |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` |  |
| OpenServiceMesh.image.tag | string | `"v0.6.1"` |  |
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
| OpenServiceMesh.webhookConfigNamePrefix | string | `"osm-webhook"` |  |


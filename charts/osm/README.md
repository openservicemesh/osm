# osm

![Version: 0.2.0](https://img.shields.io/badge/Version-0.2.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.3.0](https://img.shields.io/badge/AppVersion-v0.3.0-informational?style=flat-square)

A Helm chart to install the OSM control plane on Kubernetes

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| OpenServiceMesh.caBundleSecretName | string | `"osm-ca-bundle"` |  |
| OpenServiceMesh.certficateManager | string | `"tresor"` |  |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager"` |  |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` |  |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` |  |
| OpenServiceMesh.deployJaeger | bool | `true` |  |
| OpenServiceMesh.enableBackpressureExperimental | bool | `false` |  |
| OpenServiceMesh.enableDebugServer | bool | `false` |  |
| OpenServiceMesh.enableEgress | bool | `false` |  |
| OpenServiceMesh.enableMetricsStack | bool | `true` |  |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` |  |
| OpenServiceMesh.envoyLogLevel | string | `"debug"` |  |
| OpenServiceMesh.grafana.port | int | `3000` |  |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` |  |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` |  |
| OpenServiceMesh.image.tag | string | `"v0.3.0"` |  |
| OpenServiceMesh.imagePullSecrets | object | `{}` |  |
| OpenServiceMesh.meshCIDRRanges | string | `"0.0.0.0/0"` |  |
| OpenServiceMesh.meshName | string | `"osm"` |  |
| OpenServiceMesh.prometheus.port | int | `7070` |  |
| OpenServiceMesh.prometheus.retention.time | string | `"15d"` |  |
| OpenServiceMesh.replicaCount | int | `1` |  |
| OpenServiceMesh.serviceCertValidityMinutes | int | `1` |  |
| OpenServiceMesh.sidecarImage | string | `"envoyproxy/envoy-alpine:v1.15.0"` |  |
| OpenServiceMesh.useHTTPSIngress | bool | `false` |  |
| OpenServiceMesh.vault.host | string | `nil` |  |
| OpenServiceMesh.vault.protocol | string | `"http"` |  |
| OpenServiceMesh.vault.role | string | `"openservicemesh"` |  |
| OpenServiceMesh.vault.token | string | `nil` |  |
| OpenServiceMesh.tracing.address | string | `"jaeger.osm-system.svc.cluster.local"` |  |
| OpenServiceMesh.tracing.enable | bool | `false` |  |
| OpenServiceMesh.tracing.endpoint | string | `"/api/v2/spans"` |  |
| OpenServiceMesh.tracing.port | int | `9411` |  |

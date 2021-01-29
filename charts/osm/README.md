# Open Service Mesh Helm Chart

![Version: 0.6.1](https://img.shields.io/badge/Version-0.6.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.6.1](https://img.shields.io/badge/AppVersion-v0.6.1-informational?style=flat-square)

A Helm chart to install the [OSM](https://github.com/openservicemesh/osm) control plane on Kubernetes.

## Prerequisites

- Kubernetes v1.15+

## Get Repo Info

```console
helm repo add osm https://openservicemesh.github.io/osm
helm repo update
```

## Install Chart

```console
helm install [RELEASE_NAME] osm/osm
```

The command deploys `osm-controller` on the Kubernetes cluster in the default configuration.

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Chart

```console
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
helm upgrade [RELEASE_NAME] [CHART] --install
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

## Configuration

See [Customizing the Chart Before Installing](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing). To see all configurable options with detailed comments, visit the chart's [values.yaml](./values.yaml), or run these configuration commands:

```console
helm show values osm/osm
```

The following table lists the configurable parameters of the osm chart and their default values.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| OpenServiceMesh.caBundleSecretName | string | `"osm-ca-bundle"` | The Kubernetes secret to store `ca.crt` |
| OpenServiceMesh.certificateManager | string | `"tresor"` | The Certificate manager type: `tresor`, `vault` or `cert-manager` |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager"` | cert-manager issuer group |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` | cert-manager issuer kind |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` | cert-manager issuer namecert-manager issuer name |
| OpenServiceMesh.controllerLogLevel | string | `"trace"` | Controller log verbosity |
| OpenServiceMesh.deployGrafana | bool | `false` | Deploy Grafana |
| OpenServiceMesh.deployJaeger | bool | `false` | Deploy Jaeger in the OSM namespace |
| OpenServiceMesh.deployPrometheus | bool | `false` | Deploy Prometheus |
| OpenServiceMesh.enableBackpressureExperimental | bool | `false` | Enable experimental backpressure feature |
| OpenServiceMesh.enableDebugServer | bool | `false` | Enable the debug HTTP server |
| OpenServiceMesh.enableEgress | bool | `false` | Enable egress in the mesh |
| OpenServiceMesh.enableFluentbit | bool | `false` | Enable Fluentbit sidecar deployment |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` | Enable permissive traffic policy mode |
| OpenServiceMesh.enablePrometheusScraping | bool | `true` | Enable Prometheus metrics scraping on sidecar proxies |
| OpenServiceMesh.enableRoutesV2Experimental | bool | `false` | Enable experimental routes feature |
| OpenServiceMesh.enforceSingleMesh | bool | `false` | Enforce only deploying one mesh in the cluster |
| OpenServiceMesh.envoyLogLevel | string | `"error"` | Envoy log level is used to specify the level of logs collected from envoy |
| OpenServiceMesh.fluentBit.enableProxySupport | bool | `false` | Enable proxy support for FluentBit |
| OpenServiceMesh.fluentBit.httpProxy | string | `""` | HTTP Proxy url for FluentBit |
| OpenServiceMesh.fluentBit.httpsProxy | string | `""` | HTTPS Proxy url for FluentBit |
| OpenServiceMesh.fluentBit.logLevel | string | `"error"` | Log level for FluentBit |
| OpenServiceMesh.fluentBit.name | string | `"fluentbit-logger"` | luentBit Sidecar container name |
| OpenServiceMesh.fluentBit.outputPlugin | string | `"stdout"` | FluentBit Output Plugin, can be `stdout` or `azure` |
| OpenServiceMesh.fluentBit.primaryKey | string | `""` | PrimaryKey for FluentBit output plugin to Azure LogAnalytics |
| OpenServiceMesh.fluentBit.pullPolicy | string | `"IfNotPresent"` | PullPolicy for FluentBit sidecar container |
| OpenServiceMesh.fluentBit.registry | string | `"fluent"` | Registry for FluentBit sidecar container |
| OpenServiceMesh.fluentBit.tag | string | `"1.6.4"` | FluentBit sidecar image tag |
| OpenServiceMesh.fluentBit.workspaceId | string | `""` | WorkspaceId for FluentBit output plugin to Azure LogAnalytics |
| OpenServiceMesh.grafana.enableRemoteRendering | bool | `false` | Enable Remote Rendering in Grafana |
| OpenServiceMesh.grafana.port | int | `3000` | Grafana port |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` | `osm-controller` pod PullPolicy |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` | `osm-controller` image registry |
| OpenServiceMesh.image.tag | string | `"v0.6.1"` | `osm-controller` image tag |
| OpenServiceMesh.imagePullSecrets | list | `[]` | `osm-controller` image pull secret |
| OpenServiceMesh.meshName | string | `"osm"` | Name for the new control plane instance |
| OpenServiceMesh.osmNamespace | string | `""` | Optional parameter. If not specified, the release namespace is used to deploy the osm components. |
| OpenServiceMesh.outboundIPRangeExclusionList | list | `[]` | Optional parameter to specify a global list of IP ranges to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of IP ranges of the form a.b.c.d/x. |
| OpenServiceMesh.prometheus.port | int | `7070` | Prometheus port |
| OpenServiceMesh.prometheus.retention.time | string | `"15d"` | Prometheus retention time |
| OpenServiceMesh.replicaCount | int | `1` | `osm-controller` replicas |
| OpenServiceMesh.serviceCertValidityDuration | string | `"24h"` | Sets the service certificatevalidity duration |
| OpenServiceMesh.sidecarImage | string | `"envoyproxy/envoy-alpine:v1.17.0"` | Envoy sidecar image |
| OpenServiceMesh.tracing.address | string | `"jaeger.osm-system.svc.cluster.local"` | Tracing destination cluster (must contain the namespace) |
| OpenServiceMesh.tracing.enable | bool | `true` | Toggles Envoy's tracing functionality on/off for all sidecar proxies in the cluster |
| OpenServiceMesh.tracing.endpoint | string | `"/api/v2/spans"` | Destination's API or collector endpoint where the spans will be sent to |
| OpenServiceMesh.tracing.port | int | `9411` | Destination port for the listener |
| OpenServiceMesh.useHTTPSIngress | bool | `false` | Enables HTTPS ingress on the mesh |
| OpenServiceMesh.vault.host | string | `nil` | Hashicorp Vault host/service - where Vault is installed |
| OpenServiceMesh.vault.protocol | string | `"http"` | protocol to use to connect to Vault |
| OpenServiceMesh.vault.role | string | `"openservicemesh"` | Vault role to be used by Open Service Mesh |
| OpenServiceMesh.vault.token | string | `nil` | token that should be used to connect to Vault |
| OpenServiceMesh.webhookConfigNamePrefix | string | `"osm-webhook"` | Validating- and MutatingWebhookConfiguration name |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->
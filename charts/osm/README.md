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

| Parameter                                      | Description                                                                                       | Default                                 |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------- | --------------------------------------- |
| OpenServiceMesh.caBundleSecretName             | The Kubernetes secret to store `ca.crt`                                                           | `"osm-ca-bundle"`                       |     |
| OpenServiceMesh.certificateManager             | The Certificate manager type: `tresor`, `vault` or `cert-manager`                                 | `"tresor"`                              |     |
| OpenServiceMesh.certmanager.issuerGroup        | cert-manager issuer group                                                                         | `"cert-manager"`                        |     |
| OpenServiceMesh.certmanager.issuerKind         | cert-manager issuer kind                                                                          | `"Issuer"`                              |     |
| OpenServiceMesh.certmanager.issuerName         | cert-manager issuer namecert-manager issuer name                                                  | `"osm-ca"`                              |     |
| OpenServiceMesh.connectVault                   | Whether to use Hashicorp Vault                                                                    | `true`                                  |     |
| OpenServiceMesh.controllerLogLevel             | Controller log verbosity                                                                          | `"trace"`                               |     |
| OpenServiceMesh.deployGrafana                  | Deploy Grafana                                                                                    | `false`                                 |     |
| OpenServiceMesh.deployJaeger                   | Deploy Jaeger                                                                                     | `true`                                  |     |
| OpenServiceMesh.deployPrometheus               | Deploy Prometheus                                                                                 | `false`                                 |     |
| OpenServiceMesh.enableBackpressureExperimental | Enable experimental backpressure feature                                                          | `false`                                 |     |
| OpenServiceMesh.enableDebugServer              | Enable the debug HTTP server                                                                      | `false`                                 |     |
| OpenServiceMesh.enableEgress                   | Enable egress in the mesh                                                                         | `false`                                 |     |
| OpenServiceMesh.enableFluentbit                | Enable Fluentbit sidecar deployment                                                               | `false`                                 |     |
| OpenServiceMesh.enablePermissiveTrafficPolicy  | Enable permissive traffic policy mode                                                             | `false`                                 |     |
| OpenServiceMesh.enablePrometheusScraping       | Enable Prometheus metrics scraping on sidecar proxies                                             | `true`                                  |     |
| OpenServiceMesh.enforceSingleMesh              | Enforce only deploying one mesh in the cluster                                                    | `false`                                 |     |
| OpenServiceMesh.envoyLogLevel                  | Envoy log level is used to specify the level of logs collected from envoy                         | `"error"`                               |     |
| OpenServiceMesh.fluentBit.enableProxySupport   | Enable proxy support for FluentBit                                                                | `false`                                 |     |
| OpenServiceMesh.fluentBit.httpProxy            | HTTP Proxy url for FluentBit                                                                      | `""`                                    |     |
| OpenServiceMesh.fluentBit.httpsProxy           | HTTPS Proxy url for FluentBit                                                                     | `""`                                    |     |
| OpenServiceMesh.fluentBit.logLevel             | Log level for FluentBit                                                                           | `"error"`                               |     |
| OpenServiceMesh.fluentBit.name                 | FluentBit Sidecar container name                                                                  | `"fluentbit-logger"`                    |     |
| OpenServiceMesh.fluentBit.outputPlugin         | FluentBit Output Plugin, can be `stdout` or `azure`                                               | `"stdout"`                              |     |
| OpenServiceMesh.fluentBit.primaryKey           | PrimaryKey for FluentBit output plugin to Azure LogAnalytics                                      | `""`                                    |     |
| OpenServiceMesh.fluentBit.pullPolicy           | PullPolicy for FluentBit sidecar container                                                        | `"IfNotPresent"`                        |     |
| OpenServiceMesh.fluentBit.registry             | Registry for FluentBit sidecar container                                                          | `"fluent"`                              |     |
| OpenServiceMesh.fluentBit.tag                  | FluentBit sidecar image tag                                                                       | `"1.6.4"`                               |     |
| OpenServiceMesh.fluentBit.workspaceId          | WorkspaceId for FluentBit output plugin to Azure LogAnalytics                                     | `""`                                    |     |
| OpenServiceMesh.grafana.enableRemoteRendering  | Enable Remote Rendering in Grafana                                                                | `false`                                 |     |
| OpenServiceMesh.grafana.port                   | Grafana port                                                                                      | `3000`                                  |     |
| OpenServiceMesh.image.pullPolicy               | `osm-controller` pod PullPolicy                                                                   | `"IfNotPresent"`                        |     |
| OpenServiceMesh.image.registry                 | `osm-controller` image registry                                                                   | `"openservicemesh"`                     |     |
| OpenServiceMesh.image.tag                      | `osm-controller` image tag                                                                        | `"v0.6.1"`                              |     |
| OpenServiceMesh.imagePullSecrets               | `osm-controller` image pull secret                                                                | `[]`                                    |     |
| OpenServiceMesh.meshName                       | Name for the new control plane instance                                                           | `"osm"`                                 |     |
| OpenServiceMesh.osmNamespace                   | Optional parameter. If not specified, the release namespace is used to deploy the osm components. | `""`                                    |     |
| OpenServiceMesh.prometheus.port                | Prometheus port                                                                                   | `7070`                                  |     |
| OpenServiceMesh.prometheus.retention.time      | Prometheus retention time                                                                         | `"15d"`                                 |     |
| OpenServiceMesh.replicaCount                   | `osm-controller` replicas                                                                         | `1`                                     |     |
| OpenServiceMesh.serviceCertValidityDuration    | Sets the service certificatevalidity duration                                                     | `"24h"`                                 |     |
| OpenServiceMesh.sidecarImage                   | Envoy sidecar image                                                                               | `"envoyproxy/envoy-alpine:v1.17.0"`     |     |
| OpenServiceMesh.tracing.address                | Tracing destination cluster (must contain the namespace)                                          | `"jaeger.osm-system.svc.cluster.local"` |     |
| OpenServiceMesh.tracing.enable                 | Toggles Envoy's tracing functionality on/off for all sidecar proxies in the cluster               | `true`                                  |     |
| OpenServiceMesh.tracing.endpoint               | Destination's API or collector endpoint where the spans will be sent to                           | `"/api/v2/spans"`                       |     |
| OpenServiceMesh.tracing.port                   | Destination port for the listener                                                                 | `9411`                                  |     |
| OpenServiceMesh.useHTTPSIngress                | Enables HTTPS ingress on the mesh                                                                 | `false`                                 |     |
| OpenServiceMesh.vault.host                     | Hashicorp Vault host/service - where Vault is installed                                           | `nil`                                   |     |
| OpenServiceMesh.vault.protocol                 | protocol to use to connect to Vault                                                               | `"http"`                                |     |
| OpenServiceMesh.vault.role                     | Vault role to be used by Open Service Mesh                                                        | `"openservicemesh"`                     |     |
| OpenServiceMesh.vault.token                    | token that should be used to connect to Vault                                                     | `nil`                                   |     |
| OpenServiceMesh.webhookConfigNamePrefix        | Validating- and MutatingWebhookConfiguration name                                                 | `"osm-webhook"`                         |     |

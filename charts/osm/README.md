# Open Service Mesh Helm Chart

![Version: 0.9.0](https://img.shields.io/badge/Version-0.9.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.9.1](https://img.shields.io/badge/AppVersion-v0.9.1-informational?style=flat-square)

A Helm chart to install the [OSM](https://github.com/openservicemesh/osm) control plane on Kubernetes.

## Prerequisites

- Kubernetes >= v1.19.0

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
| OpenServiceMesh.caBundleSecretName | string | `"osm-ca-bundle"` | The Kubernetes secret name to store CA bundle for the root CA used in OSM |
| OpenServiceMesh.certKeyBitSize | int | `2048` | Certificate key bit size for data plane certificates issued to workloads to communicate over mTLS |
| OpenServiceMesh.certificateManager | string | `"tresor"` | The Certificate manager type: `tresor`, `vault` or `cert-manager` |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager.io"` | cert-manager issuer group |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` | cert-manager issuer kind |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` | cert-manager issuer namecert-manager issuer name |
| OpenServiceMesh.configResyncInterval | string | `"0s"` | Sets the resync interval for regular proxy broadcast updates, set to 0s to not enforce any resync |
| OpenServiceMesh.controlPlaneTolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| OpenServiceMesh.controllerLogLevel | string | `"info"` | Controller log verbosity |
| OpenServiceMesh.crdConverter.podLabels | object | `{}` | CRD converter's pod labels |
| OpenServiceMesh.crdConverter.replicaCount | int | `1` | CRD converter's replica count |
| OpenServiceMesh.crdConverter.resource | object | `{"limits":{"cpu":"0.5","memory":"64M"},"requests":{"cpu":"0.3","memory":"64M"}}` | CRD converter's container resource parameters |
| OpenServiceMesh.deployGrafana | bool | `false` | Deploy Grafana with OSM installation |
| OpenServiceMesh.deployJaeger | bool | `false` | Deploy Jaeger during OSM installation |
| OpenServiceMesh.deployPrometheus | bool | `false` | Deploy Prometheus with OSM installation |
| OpenServiceMesh.enableDebugServer | bool | `false` | Enable the debug HTTP server on OSM controller |
| OpenServiceMesh.enableEgress | bool | `false` | Enable egress in the mesh |
| OpenServiceMesh.enableFluentbit | bool | `false` | Enable Fluent Bit sidecar deployment on OSM controller's pod |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` | Enable permissive traffic policy mode |
| OpenServiceMesh.enablePrivilegedInitContainer | bool | `false` | Run init container in privileged mode |
| OpenServiceMesh.enforceSingleMesh | bool | `false` | Enforce only deploying one mesh in the cluster |
| OpenServiceMesh.envoyLogLevel | string | `"error"` | Log level for the Envoy proxy sidecar |
| OpenServiceMesh.featureFlags.enableAsyncProxyServiceMapping | bool | `false` | Enable async proxy-service mapping |
| OpenServiceMesh.featureFlags.enableCRDConverter | bool | `false` | Enable CRD conversion webhook. When enabled, a conversion webhook will be deployed to perform API version conversions for custom resources. |
| OpenServiceMesh.featureFlags.enableEgressPolicy | bool | `true` | Enable OSM's Egress policy API. When enabled, fine grained control over Egress (external) traffic is enforced |
| OpenServiceMesh.featureFlags.enableIngressBackendPolicy | bool | `false` | Enables OSM's IngressBackend policy API. When enabled, OSM will use the IngressBackend API allow ingress traffic to mesh backends |
| OpenServiceMesh.featureFlags.enableMulticlusterMode | bool | `false` | Enable Multicluster mode. When enabled, multicluster mode will be enabled in OSM |
| OpenServiceMesh.featureFlags.enableOSMGateway | bool | `false` | Enable OSM gateway for ingress or multicluster |
| OpenServiceMesh.featureFlags.enableValidatingWebhook | bool | `false` | Enable kubernetes validating webhook |
| OpenServiceMesh.featureFlags.enableWASMStats | bool | `true` | Enable extra Envoy statistics generated by a custom WASM extension |
| OpenServiceMesh.fluentBit.enableProxySupport | bool | `false` | Enable proxy support toggle for Fluent Bit |
| OpenServiceMesh.fluentBit.httpProxy | string | `""` | Optional HTTP proxy endpoint for Fluent Bit |
| OpenServiceMesh.fluentBit.httpsProxy | string | `""` | Optional HTTPS proxy endpoint for Fluent Bit |
| OpenServiceMesh.fluentBit.name | string | `"fluentbit-logger"` | Fluent Bit sidecar container name |
| OpenServiceMesh.fluentBit.outputPlugin | string | `"stdout"` | Fluent Bit output plugin |
| OpenServiceMesh.fluentBit.primaryKey | string | `""` | Primary Key for Fluent Bit output plugin to Log Analytics |
| OpenServiceMesh.fluentBit.pullPolicy | string | `"IfNotPresent"` | PullPolicy for Fluent Bit sidecar container |
| OpenServiceMesh.fluentBit.registry | string | `"fluent"` | Registry for Fluent Bit sidecar container |
| OpenServiceMesh.fluentBit.tag | string | `"1.6.4"` | Fluent Bit sidecar image tag |
| OpenServiceMesh.fluentBit.workspaceId | string | `""` | WorkspaceId for Fluent Bit output plugin to Log Analytics |
| OpenServiceMesh.grafana.enableRemoteRendering | bool | `false` | Enable Remote Rendering in Grafana |
| OpenServiceMesh.grafana.port | int | `3000` | Grafana service's port |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` | Container image registry |
| OpenServiceMesh.image.tag | string | `"v0.9.1"` | Container image tag |
| OpenServiceMesh.imagePullSecrets | list | `[]` | `osm-controller` image pull secret |
| OpenServiceMesh.inboundPortExclusionList | list | `[]` | Specifies a global list of ports to exclude from inbound traffic interception by the sidecar proxy. If specified, must be a list of positive integers. |
| OpenServiceMesh.injector.autoScale | object | `{"enable":false,"maxReplicas":5,"minReplicas":1,"targetAverageUtilization":80}` | Auto scale configuration |
| OpenServiceMesh.injector.autoScale.enable | bool | `false` | Enable Autoscale |
| OpenServiceMesh.injector.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| OpenServiceMesh.injector.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| OpenServiceMesh.injector.autoScale.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| OpenServiceMesh.injector.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| OpenServiceMesh.injector.podLabels | object | `{}` | Sidecar injector's pod labels |
| OpenServiceMesh.injector.replicaCount | int | `1` | Sidecar injector's replica count (ignored when autoscale.enable is true) |
| OpenServiceMesh.injector.resource | object | `{"limits":{"cpu":"0.5","memory":"64M"},"requests":{"cpu":"0.3","memory":"64M"}}` | Sidecar injector's container resource parameters |
| OpenServiceMesh.injector.webhookTimeoutSeconds | int | `20` | Mutating webhook timeout |
| OpenServiceMesh.maxDataPlaneConnections | int | `0` | Sets the max data plane connections allowed for an instance of osm-controller, set to 0 to not enforce limits |
| OpenServiceMesh.meshName | string | `"osm"` | Identifier for the instance of a service mesh within a cluster |
| OpenServiceMesh.multicluster.gatewayPort | int | `14080` | The port number of the multicluster gateway service |
| OpenServiceMesh.osmController.autoScale | object | `{"enable":false,"maxReplicas":5,"minReplicas":1,"targetAverageUtilization":80}` | Auto scale configuration |
| OpenServiceMesh.osmController.autoScale.enable | bool | `false` | Enable Autoscale |
| OpenServiceMesh.osmController.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| OpenServiceMesh.osmController.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| OpenServiceMesh.osmController.autoScale.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| OpenServiceMesh.osmController.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| OpenServiceMesh.osmController.podLabels | object | `{}` | OSM controller's pod labels |
| OpenServiceMesh.osmController.replicaCount | int | `1` | OSM controller's replica count (ignored when autoscale.enable is true) |
| OpenServiceMesh.osmController.resource | object | `{"limits":{"cpu":"1.5","memory":"512M"},"requests":{"cpu":"0.5","memory":"128M"}}` | OSM controller's container resource parameters |
| OpenServiceMesh.osmNamespace | string | `""` | Namespace to deploy OSM in. If not specified, the Helm release namespace is used. |
| OpenServiceMesh.outboundIPRangeExclusionList | list | `[]` | Specifies a global list of IP ranges to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of IP ranges of the form a.b.c.d/x. |
| OpenServiceMesh.outboundPortExclusionList | list | `[]` | Specifies a global list of ports to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of positive integers. |
| OpenServiceMesh.prometheus.port | int | `7070` | Prometheus service's port |
| OpenServiceMesh.prometheus.resources | object | `{"limits":{"cpu":"1","memory":"2G"},"requests":{"cpu":"0.5","memory":"512M"}}` | Prometheus's container resource parameters |
| OpenServiceMesh.prometheus.retention | object | `{"time":"15d"}` | Prometheus data rentention configuration |
| OpenServiceMesh.prometheus.retention.time | string | `"15d"` | Prometheus data retention time |
| OpenServiceMesh.pspEnabled | bool | `false` | Run OSM with PodSecurityPolicy configured |
| OpenServiceMesh.serviceCertValidityDuration | string | `"24h"` | Service certificate validity duration for certificate issued to workloads to communicate over mTLS |
| OpenServiceMesh.sidecarImage | string | `"envoyproxy/envoy-alpine:v1.18.3"` | Envoy sidecar image |
| OpenServiceMesh.tracing.address | string | `""` | Address of the tracing collector service (must contain the namespace). When left empty, this is computed in helper template to "jaeger.<osm-namespace>.svc.cluster.local". Please override for BYO-tracing as documented in tracing.md |
| OpenServiceMesh.tracing.enable | bool | `false` | Toggles Envoy's tracing functionality on/off for all sidecar proxies in the mesh |
| OpenServiceMesh.tracing.endpoint | string | `"/api/v2/spans"` | Tracing collector's API path where the spans will be sent to |
| OpenServiceMesh.tracing.port | int | `9411` | Port of the tracing collector service |
| OpenServiceMesh.useHTTPSIngress | bool | `false` | Enable mesh-wide HTTPS ingress capability (HTTP ingress is the default) |
| OpenServiceMesh.vault.host | string | `""` | Hashicorp Vault host/service - where Vault is installed |
| OpenServiceMesh.vault.protocol | string | `"http"` | protocol to use to connect to Vault |
| OpenServiceMesh.vault.role | string | `"openservicemesh"` | Vault role to be used by Open Service Mesh |
| OpenServiceMesh.vault.token | string | `""` | token that should be used to connect to Vault |
| OpenServiceMesh.webhookConfigNamePrefix | string | `"osm-webhook"` | Prefix used in name of the webhook configuration resources |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->
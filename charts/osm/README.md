# Open Service Mesh Helm Chart

![Version: 0.11.0](https://img.shields.io/badge/Version-0.11.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: latest-main](https://img.shields.io/badge/AppVersion-latest--main-informational?style=flat-square)

A Helm chart to install the [OSM](https://github.com/openservicemesh/osm) control plane on Kubernetes.

## Prerequisites

- Kubernetes >= 1.19.0-0

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
| OpenServiceMesh.certificateProvider.certKeyBitSize | int | `2048` | Certificate key bit size for data plane certificates issued to workloads to communicate over mTLS |
| OpenServiceMesh.certificateProvider.kind | string | `"tresor"` | The Certificate manager type: `tresor`, `vault` or `cert-manager` |
| OpenServiceMesh.certificateProvider.serviceCertValidityDuration | string | `"24h"` | Service certificate validity duration for certificate issued to workloads to communicate over mTLS |
| OpenServiceMesh.certmanager.issuerGroup | string | `"cert-manager.io"` | cert-manager issuer group |
| OpenServiceMesh.certmanager.issuerKind | string | `"Issuer"` | cert-manager issuer kind |
| OpenServiceMesh.certmanager.issuerName | string | `"osm-ca"` | cert-manager issuer namecert-manager issuer name |
| OpenServiceMesh.configResyncInterval | string | `"0s"` | Sets the resync interval for regular proxy broadcast updates, set to 0s to not enforce any resync |
| OpenServiceMesh.controlPlaneTolerations | list | `[]` | Node tolerations applied to control plane pods. The specified tolerations allow pods to schedule onto nodes with matching taints. |
| OpenServiceMesh.controllerLogLevel | string | `"info"` | Controller log verbosity |
| OpenServiceMesh.deployGrafana | bool | `false` | Deploy Grafana with OSM installation |
| OpenServiceMesh.deployJaeger | bool | `false` | Deploy Jaeger during OSM installation |
| OpenServiceMesh.deployPrometheus | bool | `false` | Deploy Prometheus with OSM installation |
| OpenServiceMesh.enableDebugServer | bool | `false` | Enable the debug HTTP server on OSM controller |
| OpenServiceMesh.enableEgress | bool | `false` | Enable egress in the mesh |
| OpenServiceMesh.enableFluentbit | bool | `false` | Enable Fluent Bit sidecar deployment on OSM controller's pod |
| OpenServiceMesh.enablePermissiveTrafficPolicy | bool | `false` | Enable permissive traffic policy mode |
| OpenServiceMesh.enablePrivilegedInitContainer | bool | `false` | Run init container in privileged mode |
| OpenServiceMesh.enableReconciler | bool | `false` | Enable reconciler for OSM's CRDs and mutating webhook |
| OpenServiceMesh.enforceSingleMesh | bool | `true` | Enforce only deploying one mesh in the cluster |
| OpenServiceMesh.envoyLogLevel | string | `"error"` | Log level for the Envoy proxy sidecar. Non developers should generally never set this value. In production environments the LogLevel should be set to `error` |
| OpenServiceMesh.featureFlags.enableAsyncProxyServiceMapping | bool | `false` | Enable async proxy-service mapping |
| OpenServiceMesh.featureFlags.enableEgressPolicy | bool | `true` | Enable OSM's Egress policy API. When enabled, fine grained control over Egress (external) traffic is enforced |
| OpenServiceMesh.featureFlags.enableEnvoyActiveHealthChecks | bool | `false` | Enable Envoy active health checks |
| OpenServiceMesh.featureFlags.enableIngressBackendPolicy | bool | `true` | Enables OSM's IngressBackend policy API. When enabled, OSM will use the IngressBackend API allow ingress traffic to mesh backends |
| OpenServiceMesh.featureFlags.enableMulticlusterMode | bool | `false` | Enable Multicluster mode. When enabled, multicluster mode will be enabled in OSM |
| OpenServiceMesh.featureFlags.enableRetryPolicy | bool | `false` | Enable Retry Policy for automatic request retries |
| OpenServiceMesh.featureFlags.enableSnapshotCacheMode | bool | `false` | Enables SnapshotCache feature for Envoy xDS server. |
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
| OpenServiceMesh.image.digest | object | `{"osmBootstrap":"sha256:79f4f27d7b7852127391fb66d8977b0569f16da76d840710f01373be665042c8","osmCRDs":"sha256:7b44a20de1dac154e9294d313e715c1b327ef15771ffe8b1a775ed83e8e7170c","osmController":"sha256:288dd6ab3095e4bcd514e12b0ef88373a335d31c1a640130541cf5ba75144d80","osmInjector":"sha256:30e70bdf0a1d42aa0f339e1925c98bbfc89a0fdd1f03c6f6fb43c2cf6cf1ff93","osmSidecarInit":"sha256:ad7315b532825e1a6b703d6664f3fd895b64a6a3758b9f738eb669d72324619a"}` | Image digest (ignored when image.tag is set) |
| OpenServiceMesh.image.digest.osmBootstrap | string | `"sha256:79f4f27d7b7852127391fb66d8977b0569f16da76d840710f01373be665042c8"` | osm-boostrap's image digest |
| OpenServiceMesh.image.digest.osmCRDs | string | `"sha256:7b44a20de1dac154e9294d313e715c1b327ef15771ffe8b1a775ed83e8e7170c"` | osm-crds' image digest |
| OpenServiceMesh.image.digest.osmController | string | `"sha256:288dd6ab3095e4bcd514e12b0ef88373a335d31c1a640130541cf5ba75144d80"` | osm-controller's image digest |
| OpenServiceMesh.image.digest.osmInjector | string | `"sha256:30e70bdf0a1d42aa0f339e1925c98bbfc89a0fdd1f03c6f6fb43c2cf6cf1ff93"` | osm-injector's image digest |
| OpenServiceMesh.image.digest.osmSidecarInit | string | `"sha256:ad7315b532825e1a6b703d6664f3fd895b64a6a3758b9f738eb669d72324619a"` | Sidecar init container's image digest |
| OpenServiceMesh.image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy for control plane containers |
| OpenServiceMesh.image.registry | string | `"openservicemesh"` | Container image registry for control plane images |
| OpenServiceMesh.image.tag | string | `""` | Container image tag for control plane images (must not be set on release branches) |
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
| OpenServiceMesh.multicluster | object | `{"gatewayLogLevel":"error"}` | OSM multicluster feature configuration |
| OpenServiceMesh.multicluster.gatewayLogLevel | string | `"error"` | Log level for the multicluster gateway |
| OpenServiceMesh.osmBootstrap.podLabels | object | `{}` | OSM bootstrap's pod labels |
| OpenServiceMesh.osmBootstrap.replicaCount | int | `1` | OSM bootstrap's replica count |
| OpenServiceMesh.osmBootstrap.resource | object | `{"limits":{"cpu":"0.5","memory":"128M"},"requests":{"cpu":"0.3","memory":"128M"}}` | OSM bootstrap's container resource parameters |
| OpenServiceMesh.osmController.autoScale | object | `{"enable":false,"maxReplicas":5,"minReplicas":1,"targetAverageUtilization":80}` | Auto scale configuration |
| OpenServiceMesh.osmController.autoScale.enable | bool | `false` | Enable Autoscale |
| OpenServiceMesh.osmController.autoScale.maxReplicas | int | `5` | Maximum replicas for autoscale |
| OpenServiceMesh.osmController.autoScale.minReplicas | int | `1` | Minimum replicas for autoscale |
| OpenServiceMesh.osmController.autoScale.targetAverageUtilization | int | `80` | Average target CPU utilization (%) |
| OpenServiceMesh.osmController.enablePodDisruptionBudget | bool | `false` | Enable Pod Disruption Budget |
| OpenServiceMesh.osmController.podLabels | object | `{}` | OSM controller's pod labels |
| OpenServiceMesh.osmController.replicaCount | int | `1` | OSM controller's replica count (ignored when autoscale.enable is true) |
| OpenServiceMesh.osmController.resource | object | `{"limits":{"cpu":"1.5","memory":"1G"},"requests":{"cpu":"0.5","memory":"128M"}}` | OSM controller's container resource parameters. See https://docs.openservicemesh.io/docs/guides/ha_scale/scale/ for more details. |
| OpenServiceMesh.osmNamespace | string | `""` | Namespace to deploy OSM in. If not specified, the Helm release namespace is used. |
| OpenServiceMesh.outboundIPRangeExclusionList | list | `[]` | Specifies a global list of IP ranges to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of IP ranges of the form a.b.c.d/x. |
| OpenServiceMesh.outboundPortExclusionList | list | `[]` | Specifies a global list of ports to exclude from outbound traffic interception by the sidecar proxy. If specified, must be a list of positive integers. |
| OpenServiceMesh.prometheus.port | int | `7070` | Prometheus service's port |
| OpenServiceMesh.prometheus.resources | object | `{"limits":{"cpu":"1","memory":"2G"},"requests":{"cpu":"0.5","memory":"512M"}}` | Prometheus's container resource parameters |
| OpenServiceMesh.prometheus.retention | object | `{"time":"15d"}` | Prometheus data rentention configuration |
| OpenServiceMesh.prometheus.retention.time | string | `"15d"` | Prometheus data retention time |
| OpenServiceMesh.sidecarImage | string | `"envoyproxy/envoy-alpine@sha256:6502a637c6c5fba4d03d0672d878d12da4bcc7a0d0fb3f1d506982dde0039abd"` | Envoy sidecar image for Linux workloads (v1.19.1) |
| OpenServiceMesh.sidecarWindowsImage | string | `"envoyproxy/envoy-windows@sha256:c904fda95891ebbccb9b1f24c1a9482c8d01cbca215dd081fc8c8db36db85f85"` | Envoy sidecar image for Windows workloads (v1.19.1) |
| OpenServiceMesh.tracing.address | string | `""` | Address of the tracing collector service (must contain the namespace). When left empty, this is computed in helper template to "jaeger.<osm-namespace>.svc.cluster.local". Please override for BYO-tracing as documented in tracing.md |
| OpenServiceMesh.tracing.enable | bool | `false` | Toggles Envoy's tracing functionality on/off for all sidecar proxies in the mesh |
| OpenServiceMesh.tracing.endpoint | string | `"/api/v2/spans"` | Tracing collector's API path where the spans will be sent to |
| OpenServiceMesh.tracing.port | int | `9411` | Port of the tracing collector service |
| OpenServiceMesh.useHTTPSIngress | bool | `false` | Enable mesh-wide HTTPS ingress capability (HTTP ingress is the default) |
| OpenServiceMesh.validatorWebhook.webhookConfigurationName | string | `""` | Name of the ValidatingWebhookConfiguration |
| OpenServiceMesh.vault.host | string | `""` | Hashicorp Vault host/service - where Vault is installed |
| OpenServiceMesh.vault.protocol | string | `"http"` | protocol to use to connect to Vault |
| OpenServiceMesh.vault.role | string | `"openservicemesh"` | Vault role to be used by Open Service Mesh |
| OpenServiceMesh.vault.token | string | `""` | token that should be used to connect to Vault |
| OpenServiceMesh.webhookConfigNamePrefix | string | `"osm-webhook"` | Prefix used in name of the webhook configuration resources |
| contour.contour | object | `{"image":{"registry":"docker.io","repository":"projectcontour/contour","tag":"v1.18.0"}}` | Contour controller configuration |
| contour.enabled | bool | `false` | Enables deployment of Contour control plane and gateway |
| contour.envoy | object | `{"image":{"registry":"docker.io","repository":"envoyproxy/envoy-alpine","tag":"v1.19.1"}}` | Contour envoy edge proxy configuration |
| smi.validateTrafficTarget | bool | `true` | Enables validation of SMI Traffic Target |

<!-- markdownlint-enable MD013 MD034 -->
<!-- markdownlint-restore -->
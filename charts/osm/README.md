osm
===
A Helm chart for to install the OSM control plane on Kubernetes

Current chart version is `0.1.0`

The OSM Command Line Interface (CLI) installs the OSM control plan into Kubernetes using this Helm chart. Alternatively, one can install the OSM control plane using this chart with the [Helm](https://helm.sh/docs/intro/install/) CLI with the following command:
```console
$ helm install osm . --namespace osm-system
```

This command is equivalent to installing the osm control plane using the OSM CLI and it is the same chart that is embedded into the OSM binary.


## Chart Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| certManager | string | `"tresor"` | Certificate manager to use (tresor or vault) |
| enablePermissiveTrafficPolicy | bool | `false` | Enable permissive traffic policy mode |
| enableDebugServer | bool | `false` | Enable the debug HTTP server |
| grafana.port | int | `3000` | Grafana port |
| image.pullPolicy | string | `"Always"` | osm-controller image pull policy |
| image.registry | string | `"smctest.azurecr.io"` |  osm-controller image registry |
| image.tag | string | `"latest"` | osm-controller image tag |
| imagePullSecrets[0].name | string | `"acr-creds"` | osm-controller image pull secrets |
| prometheus.port | int | `7070` | Prometheus port |
| prometheus.retention.time | string | `"15d"` | Prometheus retention time |
| replicaCount | int | `1` | replica count |
| serviceCertValidityMinutes | int | `1` | Duration of certificate validity in minutes |
| sidecarImage | string | `"envoyproxy/envoy-alpine:v1.14.1"` | Envoy proxy sidecar image |
| vault.host | string | `nil` | Vault host |
| vault.protocol | string | `"http"` | Vault protocol |
| vault.token | string | `nil` | Vault token |

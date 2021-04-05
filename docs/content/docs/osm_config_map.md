---
title: "OSM ConfigMap"
description: "OSM ConfigMap"
type: docs
---

# OSM ConfigMap
OSM deploys a ConfigMap `osm-config` as a part of its control plane (in the same namespace as that of the osm-controller pod) which can be updated by the mesh owner/operator at any time. The purpose of this ConfigMap is to provide the mesh owner/operator the ability to update some of the mesh configurations based on their needs. The OSM ConfigMap can be found under [charts/osm/templates](https://github.com/openservicemesh/osm/blob/release-v0.8/charts/osm/templates/osm-configmap.yaml).
To view your `osm-config` in CLI use the `kubectl get` command.
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl get configmap osm-config -n osm-system -o yaml
```

## ConfigMap Values

| Key | Chart Value |Type | Allowed Values | Default Value | Function |
|-----|-------------|------|-----------------|---------------|----------|
| egress | OpenServiceMesh.enableEgress | bool | true, false| `"false"` | Enables egress in the mesh. |
| enable_debug_server | OpenServiceMesh.enableDebugServer | bool | true, false| `"true"` | Enables a debug endpoint on the osm-controller pod to list information regarding the mesh such as proxy connections, certificates, and SMI policies. |
| enable_privileged_init_container| OpenServiceMesh.enablePrivilegedInitContainer | bool | true, false | `"false"` | Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN. |
| envoy_log_level | OpenServiceMesh.envoyLogLevel | string | trace, debug, info, warning, warn, error, critical, off | `"error"` | Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh. To update the log level for existing pods, restart the deployment with `kubectl rollout restart`. |
| max_data_plane_connections | OpenServiceMesh.maxDataPlaneConnections | int | any positive integer value | `"0"` | Sets the max data plane connections allowed for an instance of osm-controller, set to 0 to not enforce limits |
| outbound_ip_range_exclusion_list | OpenServiceMesh.outboundIPRangeExclusionList | string | comma separated list of IP ranges of the form a.b.c.d/x | `-`| Global list of IP address ranges to exclude from outbound traffic interception by the sidecar proxy. |
| permissive_traffic_policy_mode | OpenServiceMesh.enablePermissiveTrafficPolicy | bool | true, false | `"false"` | Setting to `true`, enables allow-all mode in the mesh i.e. no traffic policy enforcement in the mesh. If set to `false`, enables deny-all traffic policy in mesh i.e. an `SMI Traffic Target` is necessary for services to communicate. |
| prometheus_scraping | OpenServiceMesh.enablePrometheusScraping | bool | true, false | `"true"` | Enables Prometheus metrics scraping on sidecar proxies. |
| service_cert_validity_duration | OpenServiceMesh.serviceCertValidityDuration | string | 24h, 1h30m (any time duration) | `"24h"` | Sets the service certificate validity duration, represented as a sequence of decimal numbers each with optional fraction and a unit suffix. |
| tracing_enable | OpenServiceMesh.tracing.enable | bool | true, false | `"false"` | Enables Jaeger tracing for the mesh. |
| tracing_address | OpenServiceMesh.tracing.address | string | jaeger.mesh-namespace.svc.cluster.local | `jaeger.osm-system.svc.cluster.local` | Address of the Jaeger deployment, if tracing is enabled. |
| tracing_endpoint | OpenServiceMesh.tracing.endpoint | string | /api/v2/spans | /api/v2/spans | Endpoint for tracing data, if tracing enabled. |
| tracing_port| OpenServiceMesh.tracing.port | int | any non-zero integer value | `"9411"` | Port on which tracing is enabled. |
| use_https_ingress | OpenServiceMesh.useHTTPSIngress | bool | true, false | `"false"`| Enables HTTPS ingress on the mesh. |

## Configure OSM ConfigMap
### OSM Mesh Upgrade Command
To configure values in `osm-config` use the `osm mesh upgrade` command, so that values changed in the ConfigMap are preserved. See [here](https://github.com/openservicemesh/osm/blob/release-v0.8/cmd/cli/mesh_upgrade.go) for additional details on `osm mesh upgrade` or if you're having any issues with the command see [here](https://docs.openservicemesh.io/docs/troubleshooting/CLI/mesh_upgrade/).
```bash
osm mesh upgrade --use_https_ingress=true [--osm-namespace <namespace>]
```
> The `osm-namespace` flag can be omitted if using the default namespace.

`osm mesh upgrade` uses a pre-determined set of flags. Use the `--help` flag to get a list of flags and brief explanation of each flag.
```bash
osm mesh upgrade --help
```
 `Error: unknown flag: --<flag-name>` will occur if an incorrect flag is used.
If an invalid value is used for a flag, an error will occur that explains why the value is invalid.
Example setting boolean flag to an int:
```bash
osm mesh upgrade --enable-egress=3
Error: invalid argument "3" for "--enable-egress" flag: strconv.ParseBool: parsing "3": invalid syntax
```
#### OSM Mesh Upgrade ConfigMap Flags
| Key | Flag |Type | Default Value |
|-----|------|-----|---------------|
| egress | `--enable-egress` | bool | `"false"` |
| enable_debug_server | `--enable-debug-server` | bool | `"true"` |
| enable_privileged_init_container| `--enable-privileged-init-container` | bool | `"false"` |
| envoy_log_level | `--envoy-log-level` | string | `"error"` |
| max_data_plane_connections |`--max-data-plane-connections` | int | `"0"` |
| outbound_ip_range_exclusion_list | `--outbound-ip-range-exclusion-list` | string | `-`|
| permissive_traffic_policy_mode | `--enable-permissive-traffic-policy` | bool | `"false"` |
| prometheus_scraping |`--enable-prometheus-scraping` | bool | `"true"` |
| service_cert_validity_duration | `--service-cert-validity-duration` | string | `"24h"` |
| tracing_enable | `--enable-tracing` | bool | `"false"` |
| tracing_address | `--tracing-address` | string | `jaeger.osm-system.svc.cluster.local` |
| tracing_endpoint | `--tracing-endpoint` | string | /api/v2/spans |
| tracing_port| `--tracing-port` | int | `"9411"` |
| use_https_ingress | `--use-https-ingress` | bool | `"false"`|

### Kubectl Patch Command
To create temporary changes to `osm-config` that will not be preserved across release upgrades, we can use the `kubectl patch` command.
```bash
# Replace osm-system with osm-controller's namespace if using a non-default namespace
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"true"}}' --type=merge
```
A list of all the keys that can be changed in the ConfigMap can be found in the table [above](#configmap-values) or in the [`osm-config` yaml](https://github.com/openservicemesh/osm/blob/release-v0.8/charts/osm/templates/osm-configmap.yaml).

If an incorrect key is used, a new key-value pair will be added into `osm-config`.

If an incorrect value is used, the [validating webhook](#validating-webhook) will prevent the change with an error message explaining why the value is invalid.
For example, the below command shows what happens if we patch `use_https_ingress` to a non-boolean value.
```bash
kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"use_https_ingress":"3"}}' --type=merge
# Validating webhook will deny the change
Error from server (
use_https_ingress: must be a boolean): admission webhook "osm-config-webhook.k8s.io" denied the request:
use_https_ingress: must be a boolean
```
#### Kubectl Patch Command for Each Key Type

| Key | Type | Default Value | Kubectl Patch Command Examples |
|-----|------|---------------|--------------------------------|
| enable_debug_server | bool | `"true"` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"enable_debug_server":"false"}}' --type=merge` |
| envoy_log_level | string | `"error"` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"envoy_log_level":"info"}}' --type=merge` |
| max_data_plane_connections | int | `"0"` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"max_data_plane_connections":"1000"}}' --type=merge` |
| outbound_ip_range_exclusion_list | string | `-`| `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"outbound_ip_range_exclusion_list":"1.2.3.4/0"}}' --type=merge` |
| service_cert_validity_duration | string | `"24h"` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"service_cert_validity_duration":"2m"}}' --type=merge` |
| tracing_address | string | `jaeger.osm-system.svc.cluster.local` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"tracing_address":"1.2a.b.c3"}}' --type=merge` |
| tracing_endpoint | string | /api/v2/spans | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"tracing_endpoint":"/abracadabra"}}' --type=merge` |
| tracing_port| int | `"9411"` | `kubectl patch ConfigMap osm-config -n osm-system -p '{"data":{"tracing_port":"1234"}}' --type=merge` |

## Validating Webhook

The validating webhook validates changes made to the default fields in the `osm-config` . It prevents users who are updating configurable field values with `kubectl` commands from using values beyond the tested limits. Default fields of the osm-config can be found [below](#default-fields-in-configmap).

The validating webhook yaml can be found [here](https://github.com/openservicemesh/osm/blob/release-v0.8/charts/osm/templates/validatingwebhook.yaml) and the validation code [here](https://github.com/openservicemesh/osm/blob/release-v0.8/pkg/configurator/validating_webhook.go). The validating webhook will accept or reject the changed ConfigMap configuration for the value(s) based on the [allowed values](#configmap-values) for that field(s).

To get details on the validating webhook resource use `kubectl describe` command.
```bash
#  The validating webhook name is set as `osm-webhook-<mesh_name>`
kubectl describe validatingwebhookconfiguration osm-webhook-osm
```
### Validating Webhook Reason for Denials

| Fields | Reasons for Denial |
|--------|--------------------|
| egress | `must be a boolean` |
| enable_debug_server | `must be a boolean` |
| enable_privileged_init_container| `must be a boolean` |
| envoy_log_level | `invalid log level` |
| max_data_plane_connections | `must be a positive integer` |
| outbound_ip_range_exclusion_list | `must be a list of valid IP addresses of the form a.b.c.d/x` |
| permissive_traffic_policy_mode | `must be a boolean` |
| prometheus_scraping | `must be a boolean` |
| service_cert_validity_duration | `invalid time format must be a sequence of decimal numbers each with optional fraction and a unit suffix` |
| tracing_enable | `must be a boolean` |
| tracing_port| <ul><li>`must be an integer`</li><li>`must be between 0 and 65535`</li></ul> |
| use_https_ingress | `must be a boolean` |

> Any changes to the OSM ConfigMap metadata will be rejected with `cannot change metadata`.

### Default Fields in ConfigMap
- egress
- enable_debug_server
- enable_privileged_init_container
- envoy_log_level
- max_data_plane_connections
- permissive_traffic_policy_mode
- prometheus_scraping
- service_cert_validity_duration
- tracing_enable
- use_https_ingress

> All default fields must be present in `osm-config`, removal of one or more default fields will error with `<field>: must be included as it is a default field`.

## Troubleshooting Guide
See [here](https://docs.openservicemesh.io/docs/troubleshooting/osm_configmap/) for ConfigMap troubleshooting.

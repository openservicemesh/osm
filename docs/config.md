# OSM ConfigMap

OSM deploys a configMap `osm-config` as a part of its control plane (in the same namespace as that of the osm-controller pod) which can be updated by the mesh owner/operator at any time. The purpose of this configMap is to provide the mesh owner/operator the ability to reconfigure/update some of the mesh configurations based on their needs. 

## ConfigMap Values

| Key | Type | Possible Values | Default Value | Function |
|-----|------|-----------------|---------------|----------|
| permissive_traffic_policy_mode | bool | true, false | `"false"` | Setting to `true`, enables allow-all mode on the mesh i.e. all services in the mesh will be able to talk to one and other. If set to `false`, enables deny-all on mesh i.e. an `SMI Traffic Target` is necessary for a service in the mesh to talk to another service in the mesh. |
| egress | bool | true, false| `"false"` | Enables egress on the mesh. |
| enable_debug_server | bool | true, false| `"true"` | Enables the debug endpoint on the osm-controller pod to inspect and list most of the common structures used by the control plane at runtime. |
| envoy_log_level | string | trace, debug, info, warning, warn, error, critical, off | `"error"` | Sets the verbosity of the logging of Envoy's joining the OSM service mesh. |
| prometheus_scraping | bool | true, false | `"true"` | Enables a Prometheus listener on Envoy's which inturn enables Promethues metrics scraping in OSM. |
| service_cert_validity_duration | string | 24h, 1m, 30s (any time duration) | `"24h"` | Sets the validity duration of mesh service certificates. |
| tracing_enable | bool | true, false | `"true"` | Enables Jaeger tracing for the mesh. |
| tracing_address | string | jaeger.mesh-namespace.svc.cluster.local | `jaeger.osm-system.svc.cluster.local` | Addess of the Jaeger deployment, if tracing is enabled. |
| tracing_endpoint | string | /api/v2/spans | /api/v2/spans | Endpoint for tracing data, if tracing enabled. | 
| tracing_port| int | any non-zero integer value | `"9411"` | Port on which tracing is enabled. | 
| use_https_ingress | bool | true, false | `"false"`| Enables HTTPS ingress on the mesh. | 

---
title: "OSM ConfigMap"
description: "OSM ConfigMap"
type: docs
---

# OSM ConfigMap

OSM deploys a configMap `osm-config` as a part of its control plane (in the same namespace as that of the osm-controller pod) which can be updated by the mesh owner/operator at any time. The purpose of this configMap is to provide the mesh owner/operator the ability to update some of the mesh configurations based on their needs.

## ConfigMap Values

| Key | Chart Value |Type | Allowed Values | Default Value | Function |
|-----|-------------|------|-----------------|---------------|----------|
| permissive_traffic_policy_mode | OpenServiceMesh.enablePermissiveTrafficPolicy | bool | true, false | `"false"` | Setting to `true`, enables allow-all mode in the mesh i.e. no traffic policy enforcement in the mesh. If set to `false`, enables deny-all traffic policy in mesh i.e. an `SMI Traffic Target` is necessary for services to communicate. |
| egress | OpenServiceMesh.enableEgress | bool | true, false| `"false"` | Enables egress in the mesh. |
| enable_debug_server | OpenServiceMesh.enableDebugServer | bool | true, false| `"true"` | Enables a debug endpoint on the osm-controller pod to list information regarding the mesh such as proxy connections, certificates, and SMI policies. |
| envoy_log_level | OpenServiceMesh.envoyLogLevel | string | trace, debug, info, warning, warn, error, critical, off | `"error"` | Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh. |
| prometheus_scraping | OpenServiceMesh.enablePrometheusScraping | bool | true, false | `"true"` | Enables Prometheus metrics scraping on sidecar proxies. |
| service_cert_validity_duration | OpenServiceMesh.serviceCertValidityDuration | string | 24h, 1h30m (any time duration) | `"24h"` | Sets the service certificatevalidity duration, represented as a sequence of decimal numbers each with optional fraction and a unit suffix. |
| tracing_enable | OpenServiceMesh.tracing.enable | bool | true, false | `"true"` | Enables Jaeger tracing for the mesh. |
| tracing_address | OpenServiceMesh.tracing.address | string | jaeger.mesh-namespace.svc.cluster.local | `jaeger.osm-system.svc.cluster.local` | Address of the Jaeger deployment, if tracing is enabled. |
| tracing_endpoint | OpenServiceMesh.tracing.endpoint | string | /api/v2/spans | /api/v2/spans | Endpoint for tracing data, if tracing enabled. |
| tracing_port| OpenServiceMesh.tracing.port | int | any non-zero integer value | `"9411"` | Port on which tracing is enabled. |
| use_https_ingress | OpenServiceMesh.useHTTPSIngress | bool | true, false | `"false"`| Enables HTTPS ingress on the mesh. |
| outbound_ip_range_exclusion_list | OpenServiceMesh.outboundIPRangeExclusionList | string | comma separated list of IP ranges of the form a.b.c.d/x | `-`| Global list of IP address ranges to exclude from outbound traffic interception by the sidecar proxy. |

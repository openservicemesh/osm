# Release Notes

## Release v1.1.0

### Notable changes

- Circuit breaking support for traffic directed to in-mesh and external destinations

### Breaking changes

The following changes are not backward compatible with the previous release.

- The `osm_proxy_response_send_success_count` and `osm_proxy_response_send_error_count` metrics are now labeled with the proxy certificate's common name and XDS type, so queries to match the previous equivalent need to sum for all values of each of those labels.

### Deprecation notes

The following capabilities have been deprecated and cannot be used.

- The `osm_injector_injector_sidecar_count` and `osm_injector_injector_rq_time` metrics have been removed. The `osm_admission_webhook_response_total` and `osm_http_response_duration` metrics should be used instead.
- OSM will no longer support installation on Kubernetes version v1.19.

## Release v1.0.0

### Notable changes

- New internal control plane event management framework to handle changes to the Kubernetes cluster and policies
- Validations to reject/ignore invalid SMI TrafficTarget resources
- Control plane memory utilization improvements
- Support for TCP server-first protocols for in-mesh traffic
- Updates to Grafana dashboards to reflect accurate metrics
- OSM control plane images are now multi-architecture, built for linux/amd64 and linux/arm64

### Breaking changes

The following changes are not backward compatible with the previous release.

- Top level Helm chart keys are renamed from `OpenServiceMesh` to `osm`
- `osm mesh upgrade` no longer carries over values from previous releases. Use the `--set` flag on `osm mesh upgrade` to pass values as needed. The `--container-registry` and `--osm-image-tag` flags have also been removed in favor of `--set`.

### Deprecation notes

The following capabilities have been deprecated and cannot be used.

- Kubernetes Ingress API to configure a service mesh backend to authorize ingress traffic. OSM's IngressBackend API must be used to authorize ingress traffic between an ingress gateway and service mesh backend.

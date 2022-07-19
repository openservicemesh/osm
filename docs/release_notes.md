# Release Notes

## Release v1.2.0

### Notable changes

- Custom trust domains (i.e. certificate CommonNames) are now supported
- The authentication token used to configure the Hashicorp Vault certificate provider can now be passed in using a secretRef
- Envoy has been updated to v1.22 and uses the `envoyproxy/envoy-distroless` image instead of the deprecated `envoyproxy/envoy-alpine` image.
  - This means that `kubectl exec -c envoy ... -- sh` will no longer work for the Envoy sidecar
- Added support for Kubernetes 1.23 and 1.24
- `Rate limiting`: Added capability to perform local per-instance [rate limiting of TCP connections and HTTP requests](https://release-v1-2.docs.openservicemesh.io/docs/guides/traffic_management/rate_limiting).
- Statefulsets and headless services have been fixed and work as expected

### Breaking Changes

- The following metrics no longer use the label `common_name`, due to the fact that the common name's trust domain can rotate. Instead 2 new labels, `proxy_uuid` and `identity` have been added.
  - `osm_proxy_response_send_success_count`
  - `osm_proxy_response_send_error_count`
  - `osm_proxy_xds_request_count`
- Support for Kubernetes 1.20 and 1.21 has been dropped
- Multi-arch installation supported by the Chart Helm by customizing the `affinity` and `nodeSelector` fields
- Root service in a `TrafficSplit` configuration must have a selector matching the pods backing the leaf services. The legacy behavior where a root service without a selector matching the pods backing the leaf services is able to split traffic, has been removed.

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

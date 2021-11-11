# Release Notes

## Release v1.0.0

### Notable changes

- New internal control plane event management framework to handle changes to the Kubernetes cluster and policies
- Validations to reject/ignore invalid SMI TrafficTarget resources
- Control plane memory utilization improvements
- Support for TCP server-first protocols for in-mesh traffic
- Updates to Grafana dashboards to reflect accurate metrics

### Breaking changes

The following changes are not backward compatible with the previous release.

- Top level Helm chart keys are renamed from `OpenServiceMesh` to `osm`
- `osm mesh upgrade` no longer carries over values from previous releases. Use the `--set` flag on `osm mesh upgrade` to pass values as needed. The `--container-registry` and `--osm-image-tag` flags have also been removed in favor of `--set`.

### Deprecation notes

The following capabilities have been deprecated and cannot be used.

- Kubernetes Ingress API to configure a service mesh backend to authorize ingress traffic. OSM's IngressBackend API must be used to authorize ingress traffic between an ingress gateway and service mesh backend.

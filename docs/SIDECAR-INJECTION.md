# Sidecar Injection
Services participating in the service mesh communicate via sidecar proxies installed on pods backing the services. The following sections describe the sidecar injection workflow in OSM.

## Automatic Sidecar Injection
Automatic sidecar injection is currently the only way to inject sidecars into the service mesh. Sidecars can be automatically injected into applicable Kubernetes pods using a mutating webhook admission controller provided by OSM.

Each OSM instance is given a unique ID on installation. This ID is used while labeling namespaces as a way to configure OSM to monitor the namespaces. When a namespace is labeled with `openservicemesh.io/monitor=<mesh-name>`, pods deployed in the monitored namespaces are automatically injected with sidecars by the corresponding OSM instance.

Since sidecars are automatically injected to pods deployed in OSM monitored namespaces, pods that should not be a part of the service mesh but belong to monitored namespaces need to be explicitly annotated to disable automatic sidecar injection. Using the annotation `"openservicemesh.io/sidecar-injection": "disabled"` on the POD will inform OSM to not inject the sidecar on the POD.
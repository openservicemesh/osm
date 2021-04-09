---
title: "Application Container Startup"
description: "Application Container Startup"
type: docs
---

# Container Startup

Since OSM injects application pods that are a part of the service mesh with a sidecar proxy and sets up traffic redirection rules to route all traffic to/from pods via the sidecar proxy, in some circumstances existing application containers might not startup as expected.

## When the application container depends on network connectivity at startup

Application containers that depend on network connectivity at startup are likely to experience issues once the Envoy sidecar proxy container and the `osm-init` init container are injected into the application pod by OSM. This is because upon sidecar injection, all TCP based network traffic from application containers are routed to the sidecar proxy and subject to service mesh traffic policies. This implies that for application traffic to be routed as it would without the sidecar proxy container injected, OSM controller must first program the sidecar proxy on the application pod to allow such traffic. Without the Envoy sidecar proxy being configured, all traffic from application containers will be dropped.

When OSM is configured with permissive traffic policy mode enabled, OSM will program wildcard traffic policy rules on the Envoy sidecar proxy to allow every pod to access all services that are a part of the mesh. When OSM is configured with SMI traffic policy mode enabled, explicit SMI policies must be configured to enable communication between applications in the mesh.

Regardless of the traffic policy mode, application containers that depend on network connectivity at startup can experience problems starting up if they are not resilient to delays in the network being ready. With the Envoy proxy sidecar injected, the network is deemed ready only when the sidecar proxy has been programmed by OSM controller to allow application traffic to flow through the network.

It is recommended that application containers be resilient enough to the initial bootstrapping phase of the Envoy proxy sidecar in the application pod.

It is important to note that the [container's restart policy](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy) also influences the startup of application containers. If an application container's startup policy is set to `Never` and it depends on network connectivity to be ready at startup time, it is possible the container fails to access the network until the Envoy proxy sidecar is ready to allow the application container access to the network, thereby resulting in the application container to exit and never recover from a failed startup. For this reason, it is recommended not to use a container restart policy of `Never` if your application container depends on network connectivity at startup.

### Related issues (work in progress)

- [OSM issue 2316](https://github.com/openservicemesh/osm/issues/2316): Defer startup of application containers till the Envoy proxy sidecar is ready
- [Kubernetes issue 65502](https://github.com/kubernetes/kubernetes/issues/65502): Support startup dependencies between containers on the same pod


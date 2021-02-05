---
title: "Application Protocol Selection"
description: "Application Protocol Selection"
type: docs
---

# Application Protocol Selection

Kubernetes services expose one or more ports. A port exposed by an application running the service can serve a specific application protocol such as HTTP, TCP etc. There is a requirement to be able to determine the application protocol served by a service’s port to be able to correctly configure protocol specific configurations in Envoy (Filters, Routes, RBAC policies). Prior to Kubernetes 1.19, there is no way to determine the application protocol for a port. The IP `protocol` field on the Service resource must be one of TCP, UDP, or SCTP, so this cannot be used to determine the application protocols such HTTP, gRPC etc. To address this limitation, Kubernetes 1.19 introduced an [`AppProtocol`](https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol) field for the Service resource.

This document proposes to leverage the newly introduced `AppProtocol` field to determine the application protocol for a service’s port in OSM. The application protocol on the port will be used to infer the protocol of the inbound traffic received on that port by the upstream proxy. Based on the protocol detected, appropriate filer chains will be configured on the upstream proxy to apply protocol specific filters to handle protocol specific policies.

The `AppProtocol` can be specified by default in Kubernetes server versions >= v1.19. In older versions where this field cannot be set, the application protocol for a service port can be indicated by prefixing the protocol name as a part of the port name. If the application protocol cannot be derived,  OSM controller will use `http` as the default application protocol for a port.

*Note that for port field in the service spec, the `AppProtocol` field takes precedence over the `Name` field if both are specified.
## Example

Consider the following SMI traffic access and traffic specs policies:
- A `TCPRoute` resource named `tcp-route` that specifies the port TCP traffic should be allowed on.
- An `HTTPRouteGroup` resource named `http-route` that specifies the HTTP routes for which HTTP traffic should be allowed.
- A `TrafficTarget` resource named `test` that allows pods in the service account `sa-2` to access pods in the service account `sa-1` for the speficied TCP and HTTP rules.

```yaml
kind: TCPRoute
metadata:
  name: tcp-route
spec:
  matches:
    ports:
    - 8080
---
kind: HTTPRouteGroup
metadata:
  name: http-route
spec:
  matches:
  - name: version
    pathRegex: "/version"
    methods:
    - GET
---
kind: TrafficTarget
metadata:
  name: test
  namespace: default
spec:
  destination:
    kind: ServiceAccount
    name: sa-1 # There are 2 services under this service account:  service-1 and service-2
    namespace: default
  rules:
  - kind: TCPRoute
    name: tcp-route
  - kind: HTTPRouteGroup
    name: http-route
  sources:
  - kind: ServiceAccount
    name: sa-2
    namespace: default
```

Kubernetes service resources should explicitly specify the application protocol being served by the service's ports using the `appProtocol` field.

A service `service-1` backed by a pod in service account `sa-1` serving `http` application traffic should be defined as follows:

```yaml
kind: Service
metadata:
  name: service-1
  namespace: default
spec:
  ports:
  - port: 8080
    name: some-port
    appProtocol: http
```

A service `service-2` backed by a pod in service account `sa-1` serving raw `tcp` application traffic shold be defined as follows:

```yaml
kind: Service
metadata:
  name: service-2
  namespace: default
spec:
  ports:
  - port: 8080
    name: some-port
    appProtocol: tcp
```

If the cluster is using an older Kubernetes version where `appProtocol` cannot be specified in the Service spec, the application protocol can be indicated by prefixing the port name with the protocol as follows:

```yaml
kind: Service
metadata:
  name: service-3
  namespace: default
spec:
  ports:
  - port: 80
    name: http-someport # prefix 'http-' indicates http application protocol
  - port: 90
    name: tcp-someport # prefix 'tcp-' indicates tcp application protocol
```
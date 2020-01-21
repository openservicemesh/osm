# Service Mesh Controller Design

## Document Purpose

This document is the detailed design and architecture of the Service Mesh Controller being built in this repository.



## Overview

Service Mesh Controller (SMC) is a simple, complete, and standalone [service mesh](https://www.bing.com/search?q=What%27s+a+service+mesh%3F) solution.
SMC provides a full featured control plane. It leverages an architecture based on [Envoy](https://www.envoyproxy.io/) reverse-proxy sidecars.
While by default SMC ships with Envoy, the design utilizes [interfaces](#interfaces), which enable integrations with any service-mesh capable reverse-proxy.
SMC relies on [SMI Spec](https://smi-spec.io/) for resource declaration and configuration.
SMC ships out-of-the-box with all necessary components to deploy a complete service mesh on a single Kubernetes cluster.



## Use Case

*As on operator of services spanning diverse runtime platforms (Kubernetes and Virtual Machines on public and private clouds) I need an open source solution, which will dynamically*:
  - **Apply policies** governing HTTP access between peer services within per-URI granularity
  - **Encrypt traffic** between services leveraging mTLS and short-lived certificates with a custom CA
  - **Rotate certificates** as often as necessary to make these short-lived and remove the need for certificate revocation management
  - **Collect traces and metrics** to provide visibility into the health and operation of the services
  - **Implement traffic split** between various versions of the services deployed as defined via [SMI Spec](https://smi-spec.io/)


*The system must be:*
  - easy to understand
  - simple to install
  - effortless to maintain
  - painless to troubleshoot
  - configurable via [SMI Spec](https://smi-spec.io/)



## High-level architecture

The Service Mesh Controller project is composed of the following five high-level components:
  1. [Proxy control plane](#1-proxy-control-plane) - handles gRPC connections from the service mesh sidecar proxies
  2. [Certificate manager](#2-certificate-manager) - handles issuance and management of certificates
  3. [Endpoints provider](#3-endpoints-providers) - runtime platform observers retrieving routable IP addresses of the service mesh workloads
  4. [Mesh specification](#4-mesh-specification) - facility to retrieve SMI Spec resources
  5. [Mesh catalog](#5-mesh-catalog) - combinator of all other components


![components relationship](https://user-images.githubusercontent.com/49918230/73027022-8b030180-3e2a-11ea-8226-e466b5a68e0c.png)

([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))


## Components

Let's take a look at each component:

### (1) Proxy control plane
The proxy control plane plays a key part in operating the [service mesh](https://www.bing.com/search?q=What%27s+a+service+mesh%3F). All proxies, installed as [sidecars](https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar) establish an mTLS gRPC connection to this component and continuously receive configuration updates. This component implements the interfaces required by the specific reverse proxy chosen. Our first iteration of SMC relies on [Envoy's go-control-plane xDS](https://github.com/envoyproxy/go-control-plane).

### (2) Certificate manager
The certificate manager is component which provides each service participating in the service mesh with a TLS certificate.
These service certificates are used to establish and encrypt connections between services using mTLS.
The certificates encode the services' identity and implement the [SPIFFE X.509 Verifiable Identity Document](https://github.com/spiffe/spiffe/blob/master/standards/X509-SVID.md)

### (3) Endpoints providers
The endpoint providers are one or more components, which observer runtime platforms and resolve service names into IP addresses. These facilities understand virtual machines, virtual machine scale sets, etc. within cloud vendors, as well as pods within Kubernetes clusters.

### (4) Mesh specification
The mesh specififcation is a wrapper around the existing [SMI Spec](https://github.com/deislabs/smi-spec) componints. This component abstracts the specific storage chosen for the YAML definitions. This module is effectively a warpper around [SMI Spec's Kubernetes informers](https://github.com/deislabs/smi-sdk-go), currently abstracting away the storage (Kubernetes/etcd) specifics.

### (5) Mesh catalog
The mesh catalog is the central component of SMC, which combines the outputs of all other components into a structure, which can then be transformed to proxy configuration and dispatched to all listening proxies via the proxy control plane.
This component:
  1. Communicates with the [mesh specification module  (4)](#4-mesh-specification) to detect when a **new service** has been declared (or an existing one changed) via [SMI Spec](https://github.com/deislabs/smi-spec).
  1. Reaches out to the [certificate manager (2)](#2-certificate-manager) and requests a new TLS certificate for the newly discovered service.
  1. Retrieves the IP addresses of the mesh workloads by observing the runtime platforms via the [endpoints providers (3)](#3-endpoints-providers).
  1. Combines the outputs of 1, 2, and 3 above into a data structure, which is then passed to the [proxy control plane (1)](#1-proxy-control-plane), serialized, and sent to all relevant connected proxies.

![diagram](https://user-images.githubusercontent.com/49918230/73008758-27b3a800-3e07-11ea-894e-93f53e08731e.png)
([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))


## Detailed component description

This section outlines the conventions adopted and guiding the development of the Service Mesh Controller. Components discussed in this section:
  - (A) Proxy [sidecar](https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar) - Envoy or other reverse-proxy with service-mesh capabilities
  - (B) [Proxy Certificate](#proxy-tls-certificate) - unique X.509 certificate issued to the specific proxy
  - (C) Service - [Kubernetes service resource](https://kubernetes.io/docs/concepts/services-networking/service/) referenced in SMI Spec
  - (D) [Service Certificate](#service-tls-certificate) - X.509 certificate issued to the service
  - (E) Policy - [SMI Spec](https://smi-spec.io/) traffic policy enforced by the target service's proxy
  - Examples of service endpoints handling traffic for the given service:
    - (F) Azure VM - process running on an Azure VM, listening for connections on IP 1.2.3.11, port 81.
    - (G) Kubernetes Pod - container running on a Kubernetes cluster, listening for connections on IP 1.2.3.12, port 81.
    - (H) On-prem compute - process running on a machine within the customer's private data center, listening for connections on IP 1.2.3.13, port 81.

The Service (C) is assigned a Certificate (D), and is associated with an SMI Spec Policy (E).
Traffic for Service (C) is handled by the Endpoints (F,G,H), where each endpoint is augmented with a Proxy (A).
The Proxy (A) has a dedicated Certificate (B), which is different than the Service Cert (D), and is used for mTLS connection from the Proxy to the [proxy control plane](#1-proxy-control-plane).


![service-relationships-diagram](https://user-images.githubusercontent.com/49918230/73034530-1a191500-3e3d-11ea-9a35-a1fd8cce8b53.png)


### (C) Service
Service in the diagram above is a [Kubernetes service resource](https://kubernetes.io/docs/concepts/services-networking/service/) referenced in SMI Spec. Example is the `bookstore` service defined below and referenced by a `TrafficSplit` policy:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: bookstore
  labels:
    app: bookstore
spec:
  ports:
  - port: 80
    targetPort: 80
    name: web-port
  selector:
    app: bookstore

---

apiVersion: split.smi-spec.io/v1alpha2
kind: TrafficSplit
metadata:
  name: bookstore-traffic-split
spec:
  service: bookstore
  backends:
  - service: bookstore-v1
    weight: 100
```

### (A) Proxy
In SMC `Proxy` is defined as an abstract logical component, which:
  - fronts a mesh service process (container or binary running on Kubernetes or a VM)
  - maintains a connection to a proxy control plane (xDS server)
  - continuously receives configuration updates (xDS protocol buffers)
SMC ships out of the box with [Envoy](https://www.envoyproxy.io/) reverse-proxy implementation.

### (F,G,H) Endpoint
Within the SMC codebase `Endpoint` is defined as the IP address and port number tuple of a container or a virtual machine, which is hosting a proxy, which is fronting a process, which is a member of a service and as such participates in the service mesh.
The [service endpoints (F,G,H)](#service-endpoint) are the actual binaries serving traffic for the service (C).
An endpoint uniquely identifies a container, binary, or a process.
It has an IP address, port number, and belongs to a service.
An endpoint must belong to a single service.
A service may have zero, one or many endpoints.
Each endpoint is given a proxy, which makes it a member of the service mesh.
Each proxy can serve a single service.
A service may be served by a single proxy (one-to-one relationship).

### (D) Service TLS certificate
The service TLS certificate is a SPIFFE-compatible X.509 certificate, issued for a Service.
Proxies, fronting endpoints, which form a given service will share the certificate for the given service.
This certificate is used to establish mTLS connection with peer proxies fronting endpoints of **other services** within the service mesh.
The service certificate is short-lived.
Each service certificate's lifetime will be [approximately 48 hours](#certificate-lifetime), which eliminates the need for a certificate revocation facility.
SMC declares a type `ServiceCertificate` for these certificates.
`ServiceCertificate` is how this kind of certificate is referred to in the [Interfaces](#interfaces) section of this document.
The service certificates use the SPIFFE format for interoperability with other systems.

### (B) Proxy TLS certificate
The proxy TLS certificate is a X.509 certificate issued to each individual proxy, by the certificate manager.
This kind of certificate is different than the service certificate and is used exclusively for proxy-to-control-plane mTLS communication.
Each Envoy proxy will bootstrapped with a proxy certificate, which will be used for the xDS mTLS communication.
This kind of certificate is different than the one issued for service-to-service mTLS communication.
SMC declares a type `ProxyCertificate` for these certificates.
We refer to these certificates as `ProxyCertificate` in the [interfaces](#interfaces) declarations section of this document.
This certificate's Common Name leverages the DNS-1123 standard with the following format: `<proxy-UUID>.<service-name>`. The chosen format allows us to uniquely identify the connected proxy (`proxy-UUID`) and the service, which this proxy belongs to (`service-name`).

### (C) Policy
The policy component referenced in the diagram above (C) is any SMI Spec resource referencing the service (C).

### Certificate lifetime
The service certificates issued by the certificate manager are short-lived certificates, with a validity of approximately 48 hours.
The short certificate expiration eliminates the need for an explicit revocation mechanism.
Given certificate's expiration will be randomly shortened or extended from the 48 hours, in order to avoid [thundering herd problem](https://en.wikipedia.org/wiki/Thundering_herd_problem) inflicted on the underlying certificate management system. Proxy certificates on the other hand are long-lived certificates.

### Proxy Certificate, Proxy, and Endpoint relationship

  - `ProxyCertificate` is issued by SMC for a `Proxy`, which is expected to connect to the proxy control plane some time in the future. After the certificate is issued, and before the proxy connects to the proxy control plane, the certificate is in `unclaimed` state. The state of the certificate changes to `claimed` after a proxy has connected to the control plane using the certificate.
  - `Proxy` is the reverse-proxy, which attempts to connect to the proxy control plane; the `Proxy` may, or may not be allowed to connect to the proxy control plane.
  - `Endpoint` is fronted by a `Proxy`, and is a member of a `Service`. SMC may have discovered endpoints, via the [endpoints providers](#endpoints-providers), which belong to a given service, but SMC has not seen any proxies, fronting these endpoints, connect to the proxy control plane yet.


The **intersection** of the set of issued `ProxyCertificates` ∩ connected `Proxies` ∩ discovered `Endpoints` is the set of participants in the service mesh.


![service-mesh-participants](https://user-images.githubusercontent.com/49918230/73035216-3e75f100-3e3f-11ea-915d-c19eb03ecf97.png)

### Envoy proxy ID and service membership

  - Each `Proxy` is issued a unique `ProxyCertificate`, which is dedicated to xDS mTLS communication
  - `ProxyCertificate` has a per-proxy unique Subject CN, which identifies the `Proxy`
  - The `Proxy`'s service membership is determined by examining the CN FQDN (`<proxy-UUID>.<service-name>`), where service name is string following the second period in the CN of the `ProxyCertificate`
  - There is one unique `ProxyCertificate` issued to one `Proxy`, which is dedicated to one unique `Endpoint`, and all of these can belong to only one `Service`
  - A mesh `Service` however would be constructed by one or more (`ProxyCertificate` + `Proxy` + `Endpoint`) tuples


## Interfaces

This section defines the [Go Interfaces](https://golang.org/doc/effective_go.html#interfaces) needed for
the development of the Service Mesh Controller in [this repository](https://github.com/deislabs/smc).


This section adopts the following assumptions:
  - [1:1 relationship](https://en.wikipedia.org/wiki/One-to-one_(data_model)) between an [proxy](https://www.envoyproxy.io/docs/envoy/latest/intro/what_is_envoy) and an instance of a service. (No more than one service fronted by the same proxy.)
  - [1:1 relationship](https://en.wikipedia.org/wiki/One-to-one_(data_model)) between an [endpoint](#fgh-endpoint) (port and IP) and a [proxy](#a-proxy)


### Proxy Control Plane

The [Proxy control plane](#1-proxy-control-plane) handles gRPC connections from the service mesh sidecar proxies. This module is specialized based on the brand of reverse proxy used. The implementation details below focus on an Envoy proxy.

For a fully functional Envoy-based service mesh, the proxy control plane must implement the following 4 interfaces:
  - Cluster Discovery Service - [source](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/api/v2/cds.pb.go#L189-L194)
    ```go
    // ClusterDiscoveryServiceServer is the server API for ClusterDiscoveryService service.
    type ClusterDiscoveryServiceServer interface {
        StreamClusters(ClusterDiscoveryService_StreamClustersServer) error
        DeltaClusters(ClusterDiscoveryService_DeltaClustersServer) error
        FetchClusters(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
    }
    ```
  - Endpoint Discovery Service - [source](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/api/v2/eds.pb.go#L107-L114)
    ```go
    // EndpointDiscoveryServiceClient is the client API for EndpointDiscoveryService service.
    type EndpointDiscoveryServiceClient interface {
        StreamEndpoints(ctx context.Context, opts ...grpc.CallOption) (EndpointDiscoveryService_StreamEndpointsClient, error)
        DeltaEndpoints(ctx context.Context, opts ...grpc.CallOption) (EndpointDiscoveryService_DeltaEndpointsClient, error)
        FetchEndpoints(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
    }
    ```
  - Route Discovery Service - [source](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/api/v2/rds.pb.go#L107-L114)
    ```go
    // RouteDiscoveryServiceClient is the client API for RouteDiscoveryService service.
    type RouteDiscoveryServiceClient interface {
        StreamRoutes(ctx context.Context, opts ...grpc.CallOption) (RouteDiscoveryService_StreamRoutesClient, error)
        DeltaRoutes(ctx context.Context, opts ...grpc.CallOption) (RouteDiscoveryService_DeltaRoutesClient, error)
        FetchRoutes(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
    }
    ```
  - [Secrets Discovery Service](https://www.envoyproxy.io/docs/envoy/latest/configuration/security/secret#secret-discovery-service-sds) - [source](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/service/discovery/v2/sds.pb.go#L192-L197):
    ```go
    // SecretDiscoveryServiceServer is the server API for SecretDiscoveryService service.
    type SecretDiscoveryServiceServer interface {
        DeltaSecrets(SecretDiscoveryService_DeltaSecretsServer) error
        StreamSecrets(SecretDiscoveryService_StreamSecretsServer) error
        FetchSecrets(context.Context, *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error)
    }
    ```

#### Endpoint Discovery Service

The `StreamEndpoints` method is the entrypoint into the EDS vertical of SMC. This is
declared in the `EndpointDiscoveryServiceServer` interface, which is provided by the
[Envoy Go control plane](https://github.com/envoyproxy/go-control-plane). It is declared in [eds.pb.go](https://github.com/envoyproxy/go-control-plane/blob/7e97c9c4b2547eebdca67d672b77957f1e089c74/envoy/service/endpoint/v3alpha/eds.pb.go#L200-L205).
Methods `FetchEndpoints` and `DeltaEndpoints` are used by the EDS REST API. This project
implements gRPC only and these two methods will not be implemented.

When the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) evaluates
`StreamEndpoints` it passes a `EndpointDiscoveryService_StreamEndpointsServer` *server*. The
implementation of the `StreamEndpoints` method will then use `server.Send(response)` to send
an `envoy.DiscoveryResponce` to all connected proxies.


An [MVP](https://en.wikipedia.org/wiki/Minimum_viable_product) implementation of `StreamEndpoints`
would require:
1. a method to initialize and populate a `DiscoveryResponse` struct. This will provide connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
1. a method of notifying the system when the method described in #1 needs to be evaluated to refresh the connected Envoy proxies with the latest available endpoints

A sample implementation of `StreamEndpoints`:
```go
package smc

// StreamEndpoints updates all proxies with the list of services, endpoints, weights etc.
func (e *EDS) StreamEndpoints(server eds.EndpointDiscoveryService_StreamEndpointsServer) error {
	// Figure out what the identity of the newly connected Envoy proxy is from the client certificate.
	var ClientIdentity clientIdentity
	clientIdentity = getClientIdentity(server)

	// the EDS struct implements the proposed EndpointsDiscoverer interface
	announcementsChan chan struct{}
	announcementsChan = e.catalog.GetAnnouncementChannel()

	e.RegisterNewEndpoint()

	for {
		select {
		case <-announcementsChan
			var discoveryResponse envoy.DiscoveryResponse
			// clientIdentity is the identity of the Envoy proxy connected to this gRPC server.
			discoveryResponse = e.catalog.ListEndpoints(clientIdentity)
			eds.send(discoveryResponse)
		}
	}

	return nil
}
```

### Mesh Catalog Interface

In the previous section we proposed an implementation of the `StreamEndpoints` method. This provides
connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
The `ListEndpoints` and `GetAnnouncementChannel` methods will be provided by the SMC component, which we refer to
 as the **Mesh Catalog** in this document.

The Mesh Catalog will have access to the `MeshSpecification`, `SecretsProvider`, and the list of `EndpointsProvider`s.

```go
// MeshCatalog is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCatalog interface {

    // ListEndpoints constructs a DiscoveryResponse with all endpoints the given Envoy proxy should be aware of.
    ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, error)

    // RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
    RegisterNewEndpoint(ClientIdentity)

    // ListEndpointsProviders retrieves the full list of endpoints providers registered with Mesh Catalog.
    ListEndpointsProviders() []EndpointsProvider

    // GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
    // An event on this channel may appear as a result of a change in the SMI Sper definitions, rotation of a certificate, etc.
    GetAnnouncementChannel() chan struct{}
}
```

Additional types needed for this interface:
```go
// ClientIdentity is the assigned identity of an Envoy proxy connected to the EDS server.
// This could be certificate CN, or SPIFFE identity. (TBD)
type ClientIdentity string
```


Sample `ListEndpoints` implementation:

```go
package catalog

func (catalog *Catalog) ListEndpoints(client ClientIdentity) (envoy.DiscoveryResponse, error) {
	endpointsPerService := make(map[ServiceName][]Endpoint)
	// Iterate through all compute/cluster/cloud providers participating in the service mesh and fetch
	// lists of IP addresses and port numbers (endpoints) per service, per provider.
	for _, provider in catalog.ListEndpointsProviders() {
		for _, service in catalog.mesh.ListServices() {
			endpointsFromProviderForService := catalog.providers.ListEndpointsForService(service)
			// Merge endpointsFromProviderForService into endpointsPerService
	}
}
```

### Endpoints Providers Interface
The [Endpoints providers](#3-endpoints-providers) component provides abstractions around the Go
SDKs of various Kubernetes clusters, or cloud vendor's virtual machines and other compute, which
participate in the service mesh. Each endpoint
provider is responsible for either a particular Kubernetes cluster, or a cloud vendor subscription.
The [Mesh catalog](#5-mesh-catalog) will query each [Endpoints provider](#3-endpoints-providers) for a particular [service](#c-service).
interface).

The [Endpoints providers](#3-endpoints-providers) are aware of:
  - Kubernetes Service and their own CRD (example: `AzureResource`)
  - vendor-specific APIs and methods to retrieve IP addresses and Port numbers for Endpoints

The [Endpoints providers](#3-endpoints-providers) has no awareness of:
  - what SMI Spec is
  - what Proxy or sidecar is

> Note: As of this iteration of SMC we deliberately choose to leak the Mesh Specification implementation into the
EndpointsProvider.  The Endpoints Providers are responsible for implementing a method to
resolve an SMI-declared service to the provider's specific resource definition. For instance,
when Azure EndpointProvider's `ListEndpointsForService` is invoked with some a service name
the provider would use its own method to resolve the
service to a list of Azure URIs (example: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`).
These URIs are unique identifiers of Azure VMs, VMSS, or other compute with Envoy reverse-proxies,
participating in the service mesh.

In the sample `ListEndpoints` implementation, the Mesh Catalog loops over a list of [Endpoints providers](#3-endpoints-providers):
```go
for _, provider in catalog.ListEndpointsProviders() {
```

For each `provider` registered in the Mesh Catalog, we invoke `ListEndpointsForService`.
The function will be provided a `ServiceName`, which is an SMI-declared service. The provider will
resolve the service to its own resource ID. For example `ListEndpointsForService` invoked on the
Azure EndpointsProvider with service `webservice`, will resolve `webservice` to the URI of an
[Azure VM](https://azure.microsoft.com/en-us/services/virtual-machines/) hosting an instance of
the service: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`.
From the URI the provider will resolve the list of IP addresses of participating Envoy proxies.

```go
package smc

// EndpointsProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers.
type EndpointsProvider interface {
    // ListEndpointsForService fetches the IPs and Ports for the given service
    ListEndpointsForService(ServiceName) []Endpoint
}
```

### Mesh Specification
This component provides an abstraction around the [SMI Spec Go SDK](https://github.com/deislabs/smi-sdk-go).
The abstraction hides the Kubernetes primitives. This allows us to implement SMI Spec providers
that do not rely exclusively on Kubernetes for storage etc. Mesh Specification Interface provides
a set of methods, listing all Services, TrafficSplits, and policy definitions for the
**entire service** mesh.

The Mesh Specification implementation **has no awareness** of:
  - what Envoy or reverse-proxy is
  - what IP address, Port number, or Endpoint is
  - what Azure, Azure Resource Manager etc. is or how it works


```go
package smc

import (
    "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
    "k8s.io/api/core/v1"
)

// MeshSpecification is an interface, which provides the specification of a service mesh declared with SMI.
type MeshSpecification interface {
    // ListTrafficSplits lists TrafficSplit SMI resources.
    ListTrafficSplits() []*v1alpha2.TrafficSplit

    // ListServices fetches all services declared with SMI Spec.
    ListServices() []ServiceName

   // GetService fetches a specific Kubernetes Service referenced in an SMI Spec resource.
   GetService(ServiceName) (service *Service, exists bool, err error)
}
```



## Appendix

### Fundamental Types
The following types are referenced in the interfaces proposed in this document:

  -  Port
      ```go
      package smc

      // Port is a numerical port of an Envoy proxy
      type Port int
      ```

  -  ServiceName
      ```go
      package smc

      // ServiceName is the name of a service defined via SMI
      type ServiceName string
      ```

  -  Endpoint
      ```go
      package smc

      import "net"

      // Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
      type Endpoint struct {
          net.IP `json:"ip"`
          Port   `json:"port"`
      }
      ```

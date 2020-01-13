# Service Mesh Controller Interfaces

This document defines the [Go Interfaces](https://golang.org/doc/effective_go.html#interfaces) needed for
the development of the Service Mesh Controller in [this repository](https://github.com/deislabs/smc).

The document is written with the following assumptions in mind:
  - One-to-one relationship between an [Envoy proxy](https://www.envoyproxy.io/docs/envoy/latest/intro/what_is_envoy) and an instance of a service. (No more than one service fronted by the same Envoy.)
  - One-to-one relationship between an Endpoint (port and IP) and an Envoy proxy



## Endpoint Discovery Service
This section focuses specifically on the interfaces required to implement a fully functioning Endpoint Discovery Service for an Envoy-based service mesh.

![Diagram](https://user-images.githubusercontent.com/49918230/72295509-9ea2b100-364f-11ea-9164-362c625f4005.png)

([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))


### EDS Building Blocks
The components composing the EDS server are:
  - [Service Catalog](#service-catalog) - the heart of the service mesh controller merges the outputs of the [Mesh Topology](#mesh-topology) and [Endpoints Providers](#endpoints-providers) components. This component:
      - Keeps track of all services defined in SMI and the Endpoints serving these services
      - Maintains cache of `DiscoveryResponse` sructs sent to known Envoy proxies
  - [Mesh Topology](#mesh-topology) - a wrapper around the SMI Spec informers, abstracting away the storage that implements SMI; provides the simplest possible List* functions.
  - [Endpoints Providers](#endpoints-providers)
  - [Fundamental Types](#fundamental-types-for-smc) - supporting types like `IP`, `Port`, `ServiceName` etc.


### EDS Entrypoint
The `StreamEndpoints` function is the entrypoint into the EDS vertical of SMC. This function is
declared in the `EndpointDiscoveryServiceServer` interface, which is provided by the
[Envoy Go control plane](https://github.com/envoyproxy/go-control-plane). It is declared in [eds.pb.go](https://github.com/envoyproxy/go-control-plane/blob/7e97c9c4b2547eebdca67d672b77957f1e089c74/envoy/service/endpoint/v3alpha/eds.pb.go#L200-L205):
```go
// EndpointDiscoveryServiceServer is the server API for EndpointDiscoveryService service.
type EndpointDiscoveryServiceServer interface {
	// The resource_names field in DiscoveryRequest MeshTopologyifies a list of clusters
	// to subscribe to updates for.
	StreamEndpoints(EndpointDiscoveryService_StreamEndpointsServer) error
	DeltaEndpoints(EndpointDiscoveryService_DeltaEndpointsServer) error
	FetchEndpoints(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
}
```

Functions `FetchEndpoints` and `DeltaEndpoints` are used by the EDS REST API. This project
implements gRPC only and these two functions will not be implemented.

When the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) evaluates
`StreamEndpoints` it passes a `EndpointDiscoveryService_StreamEndpointsServer` *server*. The
implementation of the `StreamEndpoints` function will then use `server.Send(response)` to send
an `envoy.DiscoveryResponce` to all connected proxies.


An [MVP](https://en.wikipedia.org/wiki/Minimum_viable_product) implementation of `StreamEndpoints`
would require:
1. a function to initialize and populate a `DiscoveryResponse` struct. This function will provide connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
1. a method of notifying the system when the function described in #1 needs to be evaluated to refresh the connected Envoy proxies with the latest available endpoints

A simple implementation of `StreamEndpoints`:
```go
// StreamEndpoints updates all connected proxies with the list of their peers (other proxies) and the services these belong to.
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

### Service Catalog

In the previous section we proposed an implementation of the `StreamEndpoints` function. This function provides 
connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
The `ListEndpoints` and `GetAnnouncementChannel` functions will be provided by the SMC component, which we refer to
 as the **Service Catalog** in this document.

The Service Catalog will have access to the `MeshTopology`, `SecretsProvider`, and the list of `EndpointsProvider`s.

#### Interface
```go
// ServiceCatalog is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type ServiceCatalog interface {

    // ListEndpoints constructs a DescoveryResponse with all endpoints the given Envoy proxy should be aware of.
    // The bool return value indicates whether there have been any changes since the last invocation of this function.
    ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, bool, error)

    // RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
    RegisterNewEndpoint(ClientIdentity)

    // ListEndpointsProviders retrieves the full list of endpoints providers registered with Service Catalog.
    ListEndpointsProviders() []EndpointsProvider

    // GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
    GetAnnouncementChannel() chan struct{}
}
```

Additional types needed for this interface:
```go
// ClientIdentity is the assigned identity of an Envoy proxy connected to the EDS server.
// This could be certificate CN, or SPIFFE identity. (TBD)
type ClientIdentity string
```

#### Member Functions
  - `ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, bool, error)` - constructs a `DiscoveryResponse` with all endpoints the given Envoy proxy should be aware of. The function may implement caching. When no changes have been detected since the last invocation of this function, the `bool` parameter would return `true`.
  - `RegisterNewEndpoint(ClientIdentity)` - adds a newly connected Envoy proxy (new gRPC server) to the list of self-announced endpoints for a service.
  - `ListEndpointsProviders() []EndpointsProvider` - retrieves the full list of endpoints providers registered with Service Catalog so far.
  - `GetAnnouncementChannel() chan struct{}` - returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints. An event on this channel may appear as a result of a change in the SMI Sper definitions, rotation of a certificate, etc.


#### Implementation Details of the Service Catalog
A `ServiceCatalog` implementation may choose to implement:
- caching of `DiscoveryResponse` per Envoy proxy
- mapping of `ClientIdentity` and/or issued certificate to a mesh service, i.e. Envoy to service mapping

#### Sample Implementation

```go
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

### Endpoints Provider
This component provides abstractions around the Go SDKs of various Kubernetes clusters, or cloud
vendor's virtual machines and other compute, which participate in the service mesh. Each endpoint
provider is responsible for either a particular Kubernetes cluster, or a cloud vendor subscription.
The Service Catalog will query each Endpoints Provider for a particular service.
interface).

The Endpoints Providers are aware of:
  - Kubernetes Service and their own CRD (example: `AzureResource`)
  - vendor-specific APIs and methods to retrieve IP addresses and Port numbers for Endpoints

The Endpoints Provider implementation **has no awareness** of:
  - what SMI Spec is
  - what Envoy proxy is

> Note: As of this iteration of SMC we deliberately choos to leak the Mesh Topology implementation into the
EndpointsProvider.  The Endpoints Providers are responsible for implementing a method to
resolve an SMI-declared service to the provider's specific resource definition. For example,
when Azure EndpointProvider's `ListEndpointsForService` is invoked with some a service name
(`webservice` for instance), the provider would use its own method to resolve the
service to a list of Azure URIs (example: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`).
These URIs are unique identifiers of Azure VMs, VMSS, or other compute with Envoy reverse-proxies,
participating in the service mesh.

In the sample `ListEndpoints` implementation, the Service Catalog loops over a list of `EndpointsProvider`s:
```go
for _, provider in catalog.ListEndpointsProviders() {
```

For each `provider` registered in the Service Catalog, we invoke `ListEndpointsForService`.
The function will be provided a `ServiceName`, which is an SMI-declared service. The provider will
resolve the service to its own resource ID. For example `ListEndpointsForService` invoked on the
Azure EndpointsProvider with service `webservice`, will resolve `webservice` to the URI of an
[Azure VM](https://azure.microsoft.com/en-us/services/virtual-machines/) hosting an instance of
the service: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`.
From the URI the provider will resolve the list of IP addresses of participating Envoy proxies.

##### Interface:
```go
// EndpointsProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers.
type EndpointsProvider interface {
    // ListEndpointsForService fetches the IPs and Ports for the given service
    ListEndpointsForService(ServiceName) []Endpoint
}
```

### Mesh Topology
This component provides an abstraction around the [SMI Spec Go SDK](https://github.com/deislabs/smi-sdk-go).
The abstraction hides the Kubernetes primitives. This allows us to implement SMI Spec providers
that do not rely exclusively on Kubernetes for storage, cache etc. Mesh Topology Interface provides
a set of functions, listing all Services, TrafficSplits, and policy definitions for the
**entire service** mesh.

The Mesh Topology implementation **has no awareness** of:
  - what Envoy or reverse-proxy is
  - what IP address, Port number, or Endpoint is
  - what Azure, Azure Resource Manager etc. is or how it works


##### Interface:
```go
// MeshTopology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type MeshTopology interface {
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

  -  IP
      ```go
      // IP is the IP of an Envoy proxy, member of a service
      type IP string
      ```

  -  Port
      ```go
      // Port is a numerical port of an Envoy proxy
      type Port int
      ```

  -  ServiceName
      ```go
      // ServiceName is the name of a service defined via SMI
      type ServiceName string
      ```

  -  Endpoint
      ```go
      // Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
      type Endpoint struct {
          IP   `json:"ip"`
          Port `json:"port"`
      }
      ```

  -  WeightedService
      ```go
      // WeightedService is a struct of a delegated service backing a target service
      type WeightedService struct {
          ServiceName ServiceName `json:"service_name:omitempty"`
          Weight      int         `json:"weight:omitempty"`
          Endpoints   []Endpoint  `json:"endpoints:omitempty"`
      }
      ```

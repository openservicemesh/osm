# Service Mesh Controller Interfaces

This document outlines the Interfaces needed for the development of the Service Mesh Controller in this repo.
The goal of the document is to design and reach consensus before the reMeshTopologytive code is written.

Assumptions made in this document:
    - One-to-one relationship between Envoy proxy and a Service (no two services behind the same Envoy)
    - One-to-one relationship between an Endpoint (port and IP) and an Envoy proxy

## Endpoint Discovery Service

This section describes the interfaces necessary to provide fully functioning Endpoint Discovery Service for an Envoy-based service mesh.

![Diagram](https://user-images.githubusercontent.com/49918230/72006983-a675e600-3248-11ea-838b-ea6366b091f3.png)

([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))


1. The interface **already** provided by the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) (let's call this the root interface) as declared in `eds.pb.go`:
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

Functions `FetchEndpoints` and `DeltaEndpoints` are necessary for a REST API implementation of the EDS. Since this project is concerned with a gRPC implementation, these will be ignored.

When the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) evaluates the `StreamEndpoints` function it passes a `EndpointDiscoveryService_StreamEndpointsServer` *server*. The implementation of the `StreamEndpoints` function will then use `server.Send(response)` to send an `envoy.DiscoveryResponce` to all connected proxies.


For this implementation of `StreamEndpoints` we need:
1. function to initialize and populate the `DiscoveryResponse` struct. This function will provide connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
1. method notifying us when the function in 1 is to be executed

An example of what `StreamEndpoints` implementation might look like:
```go
// StreamEndpoints updates all connected proxies with the list of their peers (other proxies) and the services these belong to.
func (e *EDS) StreamEndpoints(server eds.EndpointDiscoveryService_StreamEndpointsServer) error {
	// Figure out what the identity of the newly connected Envoy proxy is from the client certificate.
	var ClientIdentity clientIdentity
	clientIdentity = getClientIdentity(server)

	// the EDS struct implements the proposed EndpointsDiscoverer interface
	announcementsChan chan struct{}
	announcementsChan = e.GetAnnouncementChannel()

	e.RegisterNewEndpoint()

	for {
		select {
		case <-announcementsChan
			var discoveryResponse envoy.DiscoveryResponse
			// clientIdentity is the identity of the Envoy proxy connected to this gRPC server.
			discoveryResponse = e.ListEndpoints(clientIdentity)
			eds.send(discoveryResponse)
		}
	}

	return nil
}
```

### Service Interface
To leverage Go's type system we already defined `ServiceName` string.
On the other hand various components of SMC may require more sophisticated tools, than just an SMI declared service.
For example the Azure provider would need a way to translate between a service name (example: `webservice`) and the URI
of an Azure virtual machine hosting an instance of the service
(example: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`).
For this reason we propose the following ServiceProvider interface:

```go
type ServiceProvider interface {
	// GetName returns the name of the service.
	GetName() ServiceName

	// ListAllowedInboundServices returns the list of services allowed to connect to this service.
	ListAllowedInboundServices() []ServiceProvider

	// GetAzureURIs returns the list of Azure URIs forming the given service.
	// Example: ["/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz",]
	ListAzureLocator() ([]AzureURI, error)

	// GetKubernetesLocator returns a list of KubernetesLocators, which are pointers to a server, namespace, service.
	// Example: [{"aks-9a31e37f.hcp.westus2.azmk8s.io", "smc", "webservice"},]
	ListKubernetesLocator() ([]KubernetesLocator, error)
}
```

### Service Catalog

In the previous section we proposed an implementation of the `StreamEndpoints` function. This function provides 
connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
The `ListEndpoints` and `GetAnnouncementChannel` functions will be provided by the SMC component, which we refer to
 as the **Service Catalog** in this document.

The Service Catalog will have access to the `MeshTopology`, `SecretsProvider`, and the list of `EndpointProvider`s.

```go
// ServiceCatalog is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type ServiceCatalog interface {

	// ListEndpoints constructs a DescoveryResponse with all endpoints the given Envoy proxy should be aware of.
	// The bool return value indicates whether there have been any changes since the last invocation of this function. 
	ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, bool, error)

 	// RegisterNewEndpoint adds a newly connected Envoy proxy to the list of self-announced endpoints for a service.
 	RegisterNewEndpoint(ClientIdentity)

	// ListEndpointProviders retrieves the full list of endpoint providers registered with Service Catalog so far.
	ListEndpointProviders() []EndpointProvider

	// RegisterEndpointProvider adds a new endpoint provider to the list within the Service Catalog.
	RegisterEndpointProvider(EndpointProvider) error

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints.
	GetAnnouncementChannel() chan struct{}
}
```

Additional types needed for this interface:
```go
// ClientIdentity is the assigned identity of an Envoy proxy connected to the EDS server.
// For example: "spiffe://some/test/example"
type ClientIdentity string
```

#### Member Functions
  - `ListEndpoints(ClientIdentity) (envoy.DiscoveryResponse, bool, error)` - constructs a `DiscoveryResponse` with all endpoints the given Envoy proxy should be aware of. The function may implement caching. When no changes have been detected since the last invocation of this function, the `bool` parameter would return `true`.
  - `RegisterNewEndpoint(ClientIdentity)` - adds a newly connected Envoy proxy (new gRPC server) to the list of self-announced endpoints for a service.
  - `ListEndpointProviders() []EndpointProvider` - retrieves the full list of endpoint providers registered with Service Catalog so far.
  - `RegisterEndpointProvider(EndpointProvider) error` - adds a new endpoint provider to the list within the Service Catalog.
  - `GetAnnouncementChannel() chan struct{}` - returns an instance of a channel, which notifies the system of an event requiring the execution of ListEndpoints. An event on this channel may appear as a result of a change in the SMI Sper definitions, rotation of a certificate, etc.


#### Implementation Details of the Service Catalog
A `ServiceCatalog` implementation may choose to implement:
- caching of `DiscoveryResponse` per Envoy proxy
- mapping of `ClientIdentity` and/or issued certificate to a mesh service, i.e. Envoy to service mapping

#### Sample Implementations
```go
func (catalog *Catalog) ListEndpoints(client ClientIdentity) (envoy.DiscoveryResponse, error) {
	endpointsPerService := make(map[ServiceName][]Endpoint)
	// Iterate through all compute/cluster/cloud providers participating in the service mesh and fetch
	// lists of IP addresses and port numbers (endpoints) per service, per provider.
	for _, provider in catalog.ListEndpointProviders() {
		for _, service in catalog.mesh.ListServices() {
			endpointsFromProviderForService := catalog.providers.ListEndpointsForService(service)
			// Merge endpointsFromProviderForService into endpointsPerService
	}
}
```

### Endpoints Providers
In the sample `ListEndpoints` implementation we loop over a list of `EndpointProvider`s:
```go
for _, provider in catalog.ListEndpointProviders() {
```
To provide the ability of composing a service mesh from multiple non-homogeneous clusters, we propose the following interfaces:

```go
// EndpointProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers.
type EndpointProvider interface {
	ListEndpointsForService(ServiceProvider) []Endpoint
	Run(stopCh <-chan struct{}) error
}
```

### Mesh Topology
This component is a wrapper around SMI and provides facilities to fetch the list of services, any traffic split policies, certificates, allowed inbound services etc.

Another set of functions is required to:
1. fetch the list of services declared via SMI
2. get the unique locators of compute hosting certain services per cloud provider

```go
// MeshTopology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type MeshTopology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []ServiceProvider
}
```

## Appendix

### Fundamental Types for SMC
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

  -  AzureURI
      ```go
      // AzureURI is a unique resource locator within Azure.
      // Example: /resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz
      type AzureURI string
      ```

  -  KubernetesLocator
      ```go
      // KubernetesLocator is a struct providing sufficient information for SMC to discover a service anywhere in the world.
      type KubernetesLocator struct {
          // ClusterID is the unique identifier of a Kubernetes cluster. For example cluster's FQDN.
          ClusterID string

          // Namespace is the namespace on the cluster, within which the service name resides.
          Namespace string

          // ServiceName is the name of the service.
          ServiceName
      }
      ```



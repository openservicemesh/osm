# Service Mesh Controller Interfaces

This document outlines the [Go Interfaces](https://golang.org/doc/effective_go.html#interfaces) needed for
the development of the Service Mesh Controller in this repo.
The goal of the document is to design and reach consensus before the reMeshTopologytive code is written.

Assumptions made in this document:
    - One-to-one relationship between Envoy proxy and a Service (no two services behind the same Envoy)
    - One-to-one relationship between an Endpoint (port and IP) and an Envoy proxy

## Endpoint Discovery Service

This section describes the interfaces necessary to provide fully functioning Endpoint Discovery Service for an Envoy-based service mesh.

![Diagram](https://user-images.githubusercontent.com/49918230/72010157-4171be80-324f-11ea-886b-e08647cc1a6a.png)

([source](https://microsoft-my.sharepoint.com/:p:/p/derayche/EZRZ-xXd06dFqlWJG5nn2wkBQCm8MMlAtRcNk6Yuir9XhA?e=zPw4FZ))


The building blocks for the proposed EDS server are:
  - [Service Catalog](#service-catalog) - the heart of the service mesh controller merges the outputs of the [Mesh Topology](#mesh-topology) and [Endpoint Providers](#endpoint-providers) components. This component:
      - Keeps track of all services defined in SMI and the Endpoints serving these services
      - Maintains cache of `DiscoveryResponse` sructs sent to known Envoy proxies
  - [Mesh Topology](#mesh-topology) - a wrapper around the SMI Spec informers, abstracting away the storage that implements SMI; provides the simplest possible List* functions.
  - [Endpoint Providers](#endpoint-providers)
  - [ServiceProvider Interface](#serviceprovider-interface) - this is an augmentation of the `ServiceName` type to provide extended functionality for consumers that need more than just a string name.
  - [Fundamental Types](#fundamental-types-for-smc) - supporting types like `IP`, `Port`, `ServiceName` etc.

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

Functions `FetchEndpoints` and `DeltaEndpoints` are used by the EDS REST API. This project implements gRPC only. These two functions will not be implemented.

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

### ServiceProvider Interface
Leveraging Go's type system we define `ServiceName` string, which allows us to refer to an SMI
declared service by name. Some components of the Service Mesh Controller may however require more
sophisticated structs than just a string service name. The Azure endpoints provider, for example,
would need a way to translate between a service name (`webservice`) and the URI of an [Azure VM](https://azure.microsoft.com/en-us/services/virtual-machines/)
hosting an instance of the service (`/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`).
For this reason we propose the following ServiceProvider interface:

```go
type ServiceProvider interface {
	// GetName returns the name of the service.
	GetName() ServiceName

	// ListAllowedInboundServices returns the list of services allowed to connect to this service.
	ListAllowedInboundServices() []ServiceProvider

	// ListAzureLocators returns the list of Azure URIs forming the given service.
	// Example: ["/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz",]
	ListAzureLocators() ([]AzureURI, error)

	// ListKubernetesLocators returns a list of KubernetesLocators, which are pointers to a server, namespace, service.
	// Example: [{"aks-9a31e37f.hcp.westus2.azmk8s.io", "smc", "webservice"},]
	ListKubernetesLocators() ([]KubernetesLocator, error)
}
```

### Service Catalog

In the previous section we proposed an implementation of the `StreamEndpoints` function. This function provides 
connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
The `ListEndpoints` and `GetAnnouncementChannel` functions will be provided by the SMC component, which we refer to
 as the **Service Catalog** in this document.

The Service Catalog will have access to the `MeshTopology`, `SecretsProvider`, and the list of `EndpointProvider`s.

#### Interface
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
// This could be certificate CN, or SPIFFE identity. (TBD)
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

#### Sample Implementation
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

### Endpoints Provider
This component provides abstractions around the Go SDKs of various Kubernetes clusters, or cloud
vendor's virtual machines and other compute, which participate in the service mesh. Each endpoint
provider is responsible for either a particular Kubernetes cluster, or a cloud vendor subscription.
The Service Catalog will query each Endpoints Provider for a particular service (`ServiceProvider`
interface).

The Endpoints Provider implementation **has no awareness** of:
  - what SMI Spec is
  - what Envoy proxy is
  - what the services participating in the mesh are

The Endpoints Provider does however expect to be queried with a `ServiceProvider` object. This
object implements a method specific to end expected by the Endpoints Provider. This method
provides mapping from a given Service Mesh ServiceName to vendor's specific compute identifier.

For instance, when `ListEndpointsForService` is invoked on an `EndpointProvider` specialied to
Azure, the `ServiceProvider` passed as an argument to `ListEndpointsForService`, would be expected
to have implemented `ListAzureLocators`. `ListAzureLocators` would return a list of Azure URIs. The
URIs are unique identifiers of Azure VMs, VMSS, or other compute with Envoy reverse-proxies,
participating in the service mesh.

In the sample `ListEndpoints` implementation, the Service Catalog loops over a list of `EndpointProvider`s:
```go
for _, provider in catalog.ListEndpointProviders() {
```

For each `provider` registered in the 

##### Interface:
    ```go
    // EndpointProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers.
    type EndpointProvider interface {
        ListEndpointsForService(ServiceProvider) []Endpoint
        Run(stopCh <-chan struct{}) error
    }
    ```

### Mesh Topology
This component provides an abstraction around the [SMI Spec Go SDK](https://github.com/deislabs/smi-sdk-go). The abstraction hides the Kubernetes primitives. This allows us to implement SMI Spec providers that does not rely exclusively on Kubernetes API, etcd etc. Mesh Topology Interface provides a set of functions, listing all Services, TrafficSplits, and policy definitions for the entire service mesh.

The Mesh Topology implementation **has no awareness** of:
  - what Envoy or reverse-proxy is
  - what IP address is
  - what Azure, Azure Resource Manager etc. is or how it works


##### Interface:
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



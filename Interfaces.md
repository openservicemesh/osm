# Service Mesh Controller Interfaces

This document outlines the Interfaces needed for the development of the Service Mesh Controller in this repo.
The goal of the document is to design and reach consensus, before the respective code is written.


## Envoy Endpoint Discovery Service

This section describes the interfaces necessary to provide fully functioning Endpoint Discovery Service for an Envoy-based service mesh.

1. The interface **already** provided by the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) (let's call this the root interface) as declared in `eds.pb.go`:
```go
// EndpointDiscoveryServiceServer is the server API for EndpointDiscoveryService service.
type EndpointDiscoveryServiceServer interface {
	// The resource_names field in DiscoveryRequest specifies a list of clusters
	// to subscribe to updates for.
	StreamEndpoints(EndpointDiscoveryService_StreamEndpointsServer) error
	DeltaEndpoints(EndpointDiscoveryService_DeltaEndpointsServer) error
	FetchEndpoints(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
}
```

Note: `FetchEndpoints` and `DeltaEndpoints` are necessary for a REST API implementation of the EDS. Since this project is concerned with a gRPC implementation, these will be ignored.

When the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) evaluates the `StreamEndpoints` function it passes a `EndpointDiscoveryService_StreamEndpointsServer` *server*. The implementation of the `StreamEndpoints` function will then use `server.Send(response)` to send an `envoy.DiscoveryResponce` to all connected proxies.


For this implementation of `StreamEndpoints` we need:
1. function to initialize and populate the `DiscoveryResponse` struct. This function will provide connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.
1. method notifying us when the function in 1 is to be executed

We propose the following interface to satisfy these requirements:

```go
// EndpointsDiscoverer is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the mesh.
type EndpointsDiscoverer interface {

	// GetEndpoints constructs a DescoveryResponse with all endpoints the given recipient should be aware of.
	GetEndpoints(recipientIdentity) (envoy.DiscoveryResponse, error)

	// GetAnnouncementChannel returns an instance of a channel, which notifies the system of an event requiring the execution of GetEndpoints.
	GetAnnouncementChannel() chan struct{}
}
```

Alternative names, following [Go interface naming convention](https://golang.org/doc/effective_go.html#interface-names): `EndpointsProvider`, `EndpointsCollector`, `EndpointsGatherer`


An example of how an implementation of the struct may be used:
```go
// StreamEndpoints updates all connected proxies with the list of their peers (other proxies) and the services these belong to.
func (e *EDS) StreamEndpoints(server eds.EndpointDiscoveryService_StreamEndpointsServer) error {

	// the EDS struct implements the proposed EndpointsDiscoverer interface
	announcementsChan chan struct{}
	announcementsChan = e.GetAnnouncementChannel()

	for {
		select {
		case <-announcementsChan
			var discoveryResponse envoy.DiscoveryResponse
			discoveryResponse = e.GetEndpoints(e.recipientIdentity)
			eds.send(discoveryResponse)
		}
	}

	return nil
}
```

# Mesh Service Discovery

In the previous section we proposed `EndpointsDiscoverer`, which is concerned with providing connected Envoy proxies with a mapping from a service name to a list of routable IP addresses and ports.

A sample implementation of

```go
func (mesh *Mesh) GetEndpoints(recipientIdentity) (envoy.DiscoveryResponse, error) {
	endpointsPerService := make(map[ServiceName][]Endpoint)
	// Iterate through all compute/cluster/cloud providers participating in the service mesh and fetch
	// lists of IP addresses and port numbers (endpoints) per service, per provider.
	for _, provider in mesh.GetComputeProviders() {
		for _, serviceName in mesh.GetServices() {
			endpointsFromProviderForService := provider.GetEndpointsForService(serviceName)
			// Merge endpointsFromProviderForService into endpointsPerService
	}
}
```

## Non-Homogeneous Clusters
To provide the ability of composing a service mesh composable of multiple non-homogeneous clusters of compute, we need two additional interfaces:

```go
// ComputeProvider is an interface to be implemented by specialized components introspecting Kubernetes, Azure, and other compute/cluster providers.
type ComputeProvide interface {
	GetEndpointsForService(svc ServiceName) []Endpoint
	Run(stopCh <-chan struct{}) error
}
```

Alternative names: `ClusterDiscoverer`, ...


## Service Mesh Interface Declarations

Another set of functions is required to:
1. fetch the list of services declared via SMI
2. get the unique locators of compute hosting certain services per cloud provider

```go
// Spec is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type Spec interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices provides a list of services declared with SMI.
	ListServices() []ServiceName

	// GetComputeIDForService returns the collection of compute platforms, which form this mesh Service.
	// For instance given a ServiceName of 'WebService', this function may return the URI of an Azure
	// VM, which is hosting this: /subscriptions/abc/resourcegroups/xyz/providers/Microsoft.Compute/virtualMachines/myVM
	GetComputeIDForService(ServiceName) ComputeID
}
```

Additional types to support `GetComputeIDForService`
```go
type AzureID string
type KubernetesID struct {
	ClusterID string
	Namespace string
	Service   string
}

type ComputeID struct {
	AzureID
	KubernetesID
}
```

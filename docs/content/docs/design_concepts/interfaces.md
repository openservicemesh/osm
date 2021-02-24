---
title: "Interfaces"
description: "Interfaces"
type: docs
weight: 6
---

## Interfaces

This section defines the [Go Interfaces](https://golang.org/doc/effective_go.html#interfaces) needed for
the development of the OSM in [this repository](https://github.com/openservicemesh/osm).

This section adopts the following assumptions:

- [1:1 relationship](<https://en.wikipedia.org/wiki/One-to-one_(data_model)>) between a [proxy](https://www.envoyproxy.io/docs/envoy/latest/intro/what_is_envoy) and an instance of a service. (No more than one service fronted by the same proxy.)
- [1:1 relationship](<https://en.wikipedia.org/wiki/One-to-one_(data_model)>) between an [endpoint](#fgh-endpoint) (port and IP) and a [proxy](#a-proxy)

### Proxy Control Plane

The [Proxy control plane](#1-proxy-control-plane) handles gRPC connections from the service mesh sidecar proxies and implements Envoy's `go-control-plane`.

For a fully functional Envoy-based service mesh, the proxy control plane must implement the following interface:

- Aggregated Discovery Service - [source](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/service/discovery/v2/ads.pb.go#L172-L176)
  ```go
  // AggregatedDiscoveryServiceServer is the server API for AggregatedDiscoveryService service.
  type AggregatedDiscoveryServiceServer interface {
  StreamAggregatedResources(AggregatedDiscoveryService_StreamAggregatedResourcesServer) error
  DeltaAggregatedResources(AggregatedDiscoveryService_DeltaAggregatedResourcesServer) error
  }
  ```

#### Aggregated Discovery Service

The `StreamAggregatedResources` method is the entrypoint into the ADS vertical of OSM. This is
declared in the `AggregatedDiscoveryServiceServer` interface, which is provided by the
[Envoy Go control plane](https://github.com/envoyproxy/go-control-plane). It is declared in [ads.pb.go](https://github.com/envoyproxy/go-control-plane/blob/e9c1190525652deb975627b2ecc3deac35714025/envoy/service/discovery/v2/ads.pb.go#L172-L176).
Method `DeltaAggregatedResources` is used by the ADS REST API. This project
implements gRPC only and this method will not be implemented.

When the [Envoy Go control plane](https://github.com/envoyproxy/go-control-plane) evaluates
`StreamAggregatedResources` it passes a `AggregatedDiscoveryService_StreamAggregatedResourcesServer` _server_. The
implementation of the `StreamAggregatedResources` method will then use `server.Send(response)` to send
an `envoy.DiscoveryResponce` to all connected proxies.

An [MVP](https://en.wikipedia.org/wiki/Minimum_viable_product) implementation of `StreamAggregatedResources`
would require:

1. Depending on the `DiscoveryRequest.TypeUrl` the `DiscoveryResponse` struct for CDS, EDS, RDS, LDS or SDS is created. This will provide connected Envoy proxies with a list of clusters, mapping of service name to list of routable IP addresses, list of permitted routes, listeners and secrets respectively.
1. a method of notifying the system when the method described in #1 needs to be evaluated to refresh the connected Envoy proxies with the latest available resources (endpoints, clusters, routes, listeners or secrets)

### Mesh Catalog Interface

In the previous section, we proposed implementation of the `StreamAggregatedResources` method. This provides
connected Envoy proxies with a list of clusters, mapping of service name to list of routable IP addresses, list of permitted routes, listeners and secrets for CDS, EDS, RDS, LDS and SDS respectively.
The `ListEndpointsForService`, `ListTrafficPolicies` methods will be provided by the OSM component, which we refer to
as the **Mesh Catalog** in this document.

The Mesh Catalog will have access to the `MeshSpec`, `CertificateManager`, and the list of `EndpointsProvider`s.

```go
// MeshCataloger is the mechanism by which the Service Mesh controller discovers all Envoy proxies connected to the catalog.
type MeshCataloger interface {
	// GetSMISpec returns the SMI spec
	GetSMISpec() smi.MeshSpec

	// ListTrafficPolicies returns all the traffic policies for a given service that Envoy proxy should be aware of.
	ListTrafficPolicies(service.MeshService) ([]trafficpolicy.TrafficTarget, error)

	// ListAllowedInboundServices lists the inbound services allowed to connect to the given service.
	ListAllowedInboundServices(service.MeshService) ([]service.MeshService, error)

	// ListAllowedInboundServiceAccounts lists the downstream service accounts that can connect to the given service account
	ListAllowedInboundServiceAccounts(service.K8sServiceAccount) ([]service.K8sServiceAccount, error)

	// ListAllowedOutboundServiceAccounts lists the upstream service accounts the given service account can connect to
	ListAllowedOutboundServiceAccounts(service.K8sServiceAccount) ([]service.K8sServiceAccount, error)

	// ListServiceAccountsForService lists the service accounts associated with the given service
	ListServiceAccountsForService(service.MeshService) ([]service.K8sServiceAccount, error)

	// ListSMIPolicies lists SMI policies.
	ListSMIPolicies() ([]*split.TrafficSplit, []service.WeightedService, []service.K8sServiceAccount, []*spec.HTTPRouteGroup, []*target.TrafficTarget)

	// ListEndpointsForService returns the list of provider endpoints corresponding to a service
	ListEndpointsForService(service.MeshService) ([]endpoint.Endpoint, error)

	// ExpectProxy catalogs the fact that a certificate was issued for an Envoy proxy and this is expected to connect to XDS.
	ExpectProxy(certificate.CommonName)

    // GetServicesFromEnvoyCertificate returns a list of services the given Envoy is a member of based on the certificate provided,
    // which is a cert issued to an Envoy for XDS communication (not Envoy-to-Envoy).
	GetServicesFromEnvoyCertificate(certificate.CommonName) ([]service.MeshService, error)

	// RegisterProxy registers a newly connected proxy with the service mesh catalog.
	RegisterProxy(*envoy.Proxy)

	// UnregisterProxy unregisters an existing proxy from the service mesh catalog
	UnregisterProxy(*envoy.Proxy)

	// GetServicesForServiceAccount returns a list of services corresponding to a service account
	GetServicesForServiceAccount(service.K8sServiceAccount) ([]service.MeshService, error)

  // GetResolvableHostnamesForUpstreamService returns the hostnames over which an upstream service is accessible from a downstream service
  // TODO : remove as a part of routes refactor (#2397)
	GetResolvableHostnamesForUpstreamService(downstream, upstream service.MeshService) ([]string, error)

	//GetWeightedClusterForService returns the weighted cluster for a service
	GetWeightedClusterForService(service service.MeshService) (service.WeightedCluster, error)

  // GetIngressRoutesPerHost returns the HTTP route matches per host associated with an ingress service
  // TODO : remove as a part of routes refactor cleanup (#2397)
  GetIngressRoutesPerHost(service.MeshService) (map[string][]trafficpolicy.HTTPRouteMatch, error)

  // GetIngressPoliciesForService returns the inbound traffic policies associated with an ingress service
  GetIngressPoliciesForService(service.MeshService, service.K8sServiceAccount) ([]*trafficpolicy.InboundTrafficPolicy, error)
}
```

Additional types needed for this interface:

```go
// MeshService is a type for a namespaced service
type MeshService struct {
	Namespace string
	Name      string
}
```

```go
// NamespacedServiceAccount is a type for a namespaced service account
type NamespacedServiceAccount struct {
	Namespace      string
	ServiceAccount string
}
```

```go
// TrafficPolicy is a struct of the allowed RoutePaths from sources to a destination
type TrafficPolicy struct {
	PolicyName       string
	Destination      TrafficResource
	Source           TrafficResource
	PolicyRoutePaths []RoutePolicy
}
```

```go
// Proxy is a representation of an Envoy proxy connected to the xDS server.
// This should at some point have a 1:1 match to an Endpoint (which is a member of a meshed service).
type Proxy struct {
	certificate.CommonName
	net.IP
	ServiceName   service.MeshService
	announcements chan announcements.Announcement

	lastSentVersion    map[TypeURI]uint64
	lastAppliedVersion map[TypeURI]uint64
	lastNonce          map[TypeURI]string
}
```

### Endpoints Providers Interface

The [Endpoints providers](#3-endpoints-providers) component provides abstractions around the Go
SDKs of various Kubernetes clusters, or cloud vendor's virtual machines and other compute, which
participate in the service mesh. Each [endpoint provider](#3-endpoints-providers) is responsible for either a particular Kubernetes cluster, or a cloud vendor subscription.
The [Mesh catalog](#5-mesh-catalog) will query each [Endpoints provider](#3-endpoints-providers) for a particular [service](#c-service), and obtain the IP addresses and ports of the endpoints handling traffic for service.

The [Endpoints providers](#3-endpoints-providers) are aware of:

- Kubernetes Service and their own CRD
- vendor-specific APIs and methods to retrieve IP addresses and Port numbers for Endpoints

The [Endpoints providers](#3-endpoints-providers) has no awareness of:

- what SMI Spec is
- what Proxy or sidecar is

> Note: As of this iteration of OSM we deliberately choose to leak the Mesh Specification implementation into the
> EndpointsProvider. The [Endpoints Providers](#3-endpoints-providers) are responsible for implementing a method to
> resolve an SMI-declared service to the provider's specific resource definition. For instance,
> when Azure EndpointProvider's `ListEndpointsForService` is invoked with some a service name
> the provider would use its own method to resolve the
> service to a list of Azure URIs (example: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`).
> These URIs are unique identifiers of Azure VMs, VMSS, or other compute with Envoy reverse-proxies,
> participating in the service mesh.

In the sample `ListEndpointsForService` implementation, the Mesh Catalog loops over a list of [Endpoints providers](#3-endpoints-providers):

```go
for _, provider := range catalog.ListEndpointsProviders() {
```

For each `provider` registered in the Mesh Catalog, we invoke `ListEndpointsForService`.
The function will be provided a `ServiceName`, which is an SMI-declared service. The provider will
resolve the service to its own resource ID. For example `ListEndpointsForService` invoked on the
Azure EndpointsProvider with service `webservice`, will resolve `webservice` to the URI of an
[Azure VM](https://azure.microsoft.com/en-us/services/virtual-machines/) hosting an instance of
the service: `/resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz`.
From the URI the provider will resolve the list of IP addresses of participating Envoy proxies.

```go
package osm

// EndpointsProvider is an interface to be implemented by components abstracting Kubernetes, Azure, and other compute/cluster providers.
type EndpointsProvider interface {
    // ListEndpointsForService fetches the IPs and Ports for the given service
    ListEndpointsForService(ServiceName) []Endpoint
}
```

### Mesh Specification

This component provides an abstraction around the [SMI Spec Go SDK](https://github.com/deislabs/smi-sdk-go).
The abstraction hides the Kubernetes primitives. This allows us to implement SMI Spec providers
that do not rely exclusively on Kubernetes for storage etc. `MeshSpec` Interface provides
a set of methods, listing all services, traffic splits, and policy definitions for the
**entire service** mesh.

The `MeshSpec` implementation **has no awareness** of:

- what Envoy or reverse-proxy is
- what IP address, Port number, or Endpoint is
- what Azure, Azure Resource Manager etc. is or how it works

```go
// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits() []*split.TrafficSplit

	// ListTrafficSplitServices lists WeightedServices for the services specified in TrafficSplit SMI resources
	ListTrafficSplitServices() []service.WeightedService

	// ListServiceAccounts lists ServiceAccount resources specified in SMI TrafficTarget resources
	ListServiceAccounts() []service.K8sServiceAccount

	// GetService fetches a Kubernetes Service resource for the given MeshService
	GetService(service.MeshService) *corev1.Service

	// ListServices Lists Kubernets Service resources that are part of monitored namespaces
	ListServices() []*corev1.Service

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// ListTrafficTargets lists SMI TrafficTarget resources
	ListTrafficTargets() []*target.TrafficTarget

	// GetBackpressurePolicy fetches the Backpressure policy for the MeshService
	GetBackpressurePolicy(service.MeshService) *backpressure.Backpressure

	// GetAnnouncementsChannel returns the channel on which SMI client makes announcements
	GetAnnouncementsChannel() <-chan interface{}
}
```

### Certificate Manager

The `certificate.Manager` as shown below is as simple as having a single method for issuing certificates, and another for obtaining a notification channel.

```go
package certificate

// Manager is the interface declaring the methods for the Certificate Manager.
type Manager interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(CommonName, time.Duration) (Certificater, error)

	// GetCertificate returns a certificate given its Common Name (CN)
	GetCertificate(CommonName) (Certificater, error)

	// RotateCertificate rotates an existing certificate.
	RotateCertificate(CommonName) (Certificater, error)

	// GetRootCertificate returns the root certificate.
	GetRootCertificate() (Certificater, error)

	// ListCertificates lists all certificates issued
	ListCertificates() ([]Certificater, error)

	// ReleaseCertificate informs the underlying certificate issuer that the given cert will no longer be needed.
	// This method could be called when a given payload is terminated. Calling this should remove certs from cache and free memory if possible.
	ReleaseCertificate(CommonName)

	// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the issued certificates.
	GetAnnouncementsChannel() <-chan interface{}
}
```

Additionally we define an interface for the `Certificate` object, which requires the following methods:

```go
// Certificater is the interface declaring methods each Certificate object must have.
type Certificater interface {

	// GetCommonName retrieves the name of the certificate.
	GetCommonName() CommonName

	// GetCertificateChain retrieves the cert chain.
	GetCertificateChain() []byte

	// GetPrivateKey returns the private key.
	GetPrivateKey() []byte

	// GetIssuingCA returns the root certificate for the given cert.
	GetIssuingCA() Certificater

	// GetExpiration() returns the expiration of the certificate
	GetExpiration() time.Time

 	// GetSerialNumber returns the serial number of the given certificate.
 	GetSerialNumber() string
}
```

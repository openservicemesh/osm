package compute

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

// Interface is an interface to be implemented by components abstracting Kubernetes, and other compute/cluster providers
type Interface interface {
	k8s.PassthroughInterface

	// GetServicesForServiceIdentity retrieves the namespaced services for a given service identity
	GetServicesForServiceIdentity(identity.ServiceIdentity) []service.MeshService

	// ListServices returns a list of services that are part of monitored namespaces
	ListServices() []service.MeshService

	// ListServiceIdentitiesForService returns service identities for given service
	ListServiceIdentitiesForService(service.MeshService) []identity.ServiceIdentity

	// ListEndpointsForService retrieves the IP addresses comprising the given service.
	ListEndpointsForService(service.MeshService) []endpoint.Endpoint

	// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
	ListEndpointsForIdentity(identity.ServiceIdentity) []endpoint.Endpoint

	// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service FQDN is resolved under
	// the scope of the provider
	GetResolvableEndpointsForService(service.MeshService) []endpoint.Endpoint

	IsMetricsEnabled(*envoy.Proxy) (bool, error)

	GetHostnamesForService(svc service.MeshService, localNamespace bool) []string

	// GetServicesForProxy gets the services that map to the given proxy.
	GetServicesForProxy(p *envoy.Proxy) ([]service.MeshService, error)
}

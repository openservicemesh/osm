package fake

import (
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Provider interface combines endpoint.Provider and service.Provider
type Provider interface {
	endpoint.Provider
	service.Provider
}

type providerOption func(c *fakeClient)

// WithDefaultDemo adds some default demo data to the provider.
func WithDefaultDemo(c *fakeClient) {
	c.endpoints[tests.BookstoreV1Service.String()] = append(c.endpoints[tests.BookstoreV1Service.String()], tests.Endpoint)
	c.endpoints[tests.BookstoreV2Service.String()] = append(c.endpoints[tests.BookstoreV2Service.String()], tests.Endpoint)
	c.endpoints[tests.BookbuyerService.String()] = append(c.endpoints[tests.BookbuyerService.String()], tests.Endpoint)
	c.endpoints[tests.BookstoreApexService.String()] = append(c.endpoints[tests.BookstoreApexService.String()], tests.Endpoint)

	c.services[tests.BookstoreServiceIdentity] = append(c.services[tests.BookstoreServiceIdentity], tests.BookstoreV1Service, tests.BookstoreApexService)
	c.services[tests.BookstoreV2ServiceIdentity] = append(c.services[tests.BookstoreV2ServiceIdentity], tests.BookstoreV2Service)
	c.services[tests.BookbuyerServiceIdentity] = append(c.services[tests.BookbuyerServiceIdentity], tests.BookbuyerService)

	c.svcAccountEndpoints[tests.BookstoreServiceIdentity] = append(c.svcAccountEndpoints[tests.BookstoreServiceIdentity], tests.Endpoint, tests.Endpoint)
	c.svcAccountEndpoints[tests.BookstoreV2ServiceIdentity] = append(c.svcAccountEndpoints[tests.BookstoreV2ServiceIdentity], tests.Endpoint)
	c.svcAccountEndpoints[tests.BookbuyerServiceIdentity] = append(c.svcAccountEndpoints[tests.BookbuyerServiceIdentity], tests.Endpoint)
}

// WithIdentityServiceMapping adds a mapping between a service and a service identity.
func WithIdentityServiceMapping(si identity.ServiceIdentity, svcs []service.MeshService) providerOption {
	return func(c *fakeClient) {
		c.services[si] = append(c.services[si], svcs...)
	}
}

// NewFakeProvider implements mesh.EndpointsProvider, which creates a test Kubernetes cluster/compute provider.
func NewFakeProvider(opts ...providerOption) Provider {
	c := &fakeClient{
		endpoints:           map[string][]endpoint.Endpoint{},
		services:            map[identity.ServiceIdentity][]service.MeshService{},
		svcAccountEndpoints: map[identity.ServiceIdentity][]endpoint.Endpoint{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type fakeClient struct {
	endpoints           map[string][]endpoint.Endpoint
	services            map[identity.ServiceIdentity][]service.MeshService
	svcAccountEndpoints map[identity.ServiceIdentity][]endpoint.Endpoint
}

// ListEndpointsForService retrieves the IP addresses comprising the given service.
func (f *fakeClient) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	return f.endpoints[svc.String()]
}

// ListEndpointsForIdentity retrieves the IP addresses comprising the given service account.
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (f *fakeClient) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	return f.svcAccountEndpoints[serviceIdentity]
}

func (f *fakeClient) GetServicesForServiceIdentity(serviceIdentity identity.ServiceIdentity) []service.MeshService {
	return f.services[serviceIdentity]
}

func (f *fakeClient) ListServices() []service.MeshService {
	var services []service.MeshService

	for _, svcs := range f.services {
		services = append(services, svcs...)
	}
	return services
}

func (f *fakeClient) ListServiceIdentitiesForService(svc service.MeshService) []identity.ServiceIdentity {
	var serviceIdentities []identity.ServiceIdentity

	for svcID := range f.services {
		serviceIdentities = append(serviceIdentities, svcID)
	}
	return serviceIdentities
}

// GetID returns the unique identifier of the Provider.
func (f *fakeClient) GetID() string {
	return "Fake Kubernetes Client"
}

func (f *fakeClient) GetResolvableEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	return f.endpoints[svc.String()]
}

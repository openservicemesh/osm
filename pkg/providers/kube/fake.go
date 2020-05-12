package kube

import (
	"fmt"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/service"
	"github.com/open-service-mesh/osm/pkg/tests"
)

// NewFakeProvider implements mesh.EndpointsProvider, which creates a test Kubernetes cluster/compute provider.
func NewFakeProvider() endpoint.Provider {

	return &fakeClient{
		endpoints: map[service.Name][]endpoint.Endpoint{
			tests.NamespacedServiceName: {tests.Endpoint},
		},
		services: map[service.NamespacedServiceAccount][]service.NamespacedService{
			tests.BookstoreServiceAccount: {tests.BookstoreService},
			tests.BookbuyerServiceAccount: {tests.BookbuyerService},
		},
	}
}

type fakeClient struct {
	endpoints map[service.Name][]endpoint.Endpoint
	services  map[service.NamespacedServiceAccount][]service.NamespacedService
}

// Retrieve the IP addresses comprising the given service.
func (f fakeClient) ListEndpointsForService(name service.Name) []endpoint.Endpoint {
	if svc, ok := f.endpoints[name]; ok {
		return svc
	}
	panic(fmt.Sprintf("You are asking for ServiceName=%s but the fake Kubernetes client has not been initialized with this. What we have is: %+v", name, f.endpoints))
}

// Retrieve the service for a given service account
func (f fakeClient) ListServicesForServiceAccount(account service.NamespacedServiceAccount) []service.NamespacedService {
	services, ok := f.services[account]
	if !ok {
		panic(fmt.Sprintf("You asked fake k8s provider's ListServicesForServiceAccount for a ServiceAccount=%s, but that's not in cache: %+v", account, f.services))
	}
	return services
}

// GetID returns the unique identifier of the EndpointsProvider.
func (f fakeClient) GetID() string {
	return "Fake Kubernetes Client"
}

// GetAnnouncementsChannel obtains the channel on which providers will announce changes to the infrastructure.
func (f fakeClient) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}

package kube

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/tests"
)

// NewFakeProvider implements mesh.EndpointsProvider, which creates a test Kubernetes cluster/compute provider.
func NewFakeProvider() endpoint.Provider {
	return &fakeClient{
		endpoints: map[string][]endpoint.Endpoint{
			tests.BookstoreV1Service.String():   {tests.Endpoint},
			tests.BookstoreV2Service.String():   {tests.Endpoint},
			tests.BookbuyerService.String():     {tests.Endpoint},
			tests.BookstoreApexService.String(): {tests.Endpoint},
		},
		services: map[service.K8sServiceAccount][]service.MeshService{
			tests.BookstoreServiceAccount: {tests.BookstoreV1Service, tests.BookstoreV2Service, tests.BookstoreApexService},
			tests.BookbuyerServiceAccount: {tests.BookbuyerService},
		},
	}
}

type fakeClient struct {
	endpoints map[string][]endpoint.Endpoint
	services  map[service.K8sServiceAccount][]service.MeshService
}

// Retrieve the IP addresses comprising the given service.
func (f fakeClient) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	if svc, ok := f.endpoints[svc.String()]; ok {
		return svc
	}
	panic(fmt.Sprintf("You are asking for MeshService=%s but the fake Kubernetes client has not been initialized with this. What we have is: %+v", svc.String(), f.endpoints))
}

func (f fakeClient) GetServicesForServiceAccount(svcAccount service.K8sServiceAccount) ([]service.MeshService, error) {
	services, ok := f.services[svcAccount]
	if !ok {
		panic(fmt.Sprintf("ServiceAccount %s is not in cache: %+v", svcAccount, f.services))
	}
	return services, nil
}

// GetID returns the unique identifier of the EndpointsProvider.
func (f fakeClient) GetID() string {
	return "Fake Kubernetes Client"
}

// GetAnnouncementsChannel obtains the channel on which providers will announce changes to the infrastructure.
func (f fakeClient) GetAnnouncementsChannel() <-chan interface{} {
	return make(chan interface{})
}

func (f fakeClient) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	endpoints, found := f.endpoints[svc.String()]
	if !found {
		return nil, errServiceNotFound
	}
	return endpoints, nil
}

package kube

import (
	"fmt"

	"github.com/pkg/errors"

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

// NewFakeProvider implements mesh.EndpointsProvider, which creates a test Kubernetes cluster/compute provider.
func NewFakeProvider() Provider {
	return fakeClient{
		endpoints: map[string][]endpoint.Endpoint{
			tests.BookstoreV1Service.String():   {tests.Endpoint},
			tests.BookstoreV2Service.String():   {tests.Endpoint},
			tests.BookbuyerService.String():     {tests.Endpoint},
			tests.BookstoreApexService.String(): {tests.Endpoint},
		},
		services: map[identity.K8sServiceAccount][]service.MeshService{
			tests.BookstoreServiceAccount:   {tests.BookstoreV1Service, tests.BookstoreApexService},
			tests.BookstoreV2ServiceAccount: {tests.BookstoreV2Service},
			tests.BookbuyerServiceAccount:   {tests.BookbuyerService},
		},
		svcAccountEndpoints: map[identity.K8sServiceAccount][]endpoint.Endpoint{
			tests.BookstoreServiceAccount:   {tests.Endpoint, tests.Endpoint},
			tests.BookstoreV2ServiceAccount: {tests.Endpoint},
			tests.BookbuyerServiceAccount:   {tests.Endpoint},
		},
	}
}

type fakeClient struct {
	endpoints           map[string][]endpoint.Endpoint
	services            map[identity.K8sServiceAccount][]service.MeshService
	svcAccountEndpoints map[identity.K8sServiceAccount][]endpoint.Endpoint
}

// ListEndpointsForService retrieves the IP addresses comprising the given service.
func (f fakeClient) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	if svc, ok := f.endpoints[svc.String()]; ok {
		return svc
	}
	panic(fmt.Sprintf("You are asking for MeshService=%s but the fake Kubernetes client has not been initialized with this. What we have is: %+v", svc, f.endpoints))
}

// ListEndpointsForIdentity retrieves the IP addresses comprising the given service account.
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (f fakeClient) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	sa := serviceIdentity.ToK8sServiceAccount()
	if ep, ok := f.svcAccountEndpoints[sa]; ok {
		return ep
	}
	panic(fmt.Sprintf("You are asking for K8sServiceAccount=%s but the fake Kubernetes client has not been initialized with this. What we have is: %+v", sa, f.svcAccountEndpoints))
}

func (f fakeClient) GetServicesForServiceIdentity(serviceIdentity identity.ServiceIdentity) ([]service.MeshService, error) {
	sa := serviceIdentity.ToK8sServiceAccount()
	services, ok := f.services[sa]
	if !ok {
		return nil, errors.Errorf("ServiceAccount %s is not in cache: %+v", sa, f.services)
	}
	return services, nil
}

func (f fakeClient) ListServices() ([]service.MeshService, error) {
	var services []service.MeshService

	for _, svcs := range f.services {
		services = append(services, svcs...)
	}
	return services, nil
}

func (f fakeClient) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	var serviceIdentities []identity.ServiceIdentity

	for svcID := range f.services {
		serviceIdentities = append(serviceIdentities, svcID.ToServiceIdentity())
	}
	return serviceIdentities, nil
}

func (f fakeClient) GetHostnamesForService(svc service.MeshService, locality service.Locality) []string {
	var domains []string

	serviceName := svc.Name
	namespace := svc.Namespace

	// Referencing a local service in the local namespace
	if locality == service.LocalNS && svc.Local() {
		// Within the same namespace, service name is resolvable to its address
		domains = append(domains, serviceName) // service
	}

	// Referencing a local service in the local cluster
	if svc.Local() && locality != service.RemoteCluster {
		domains = append(domains, fmt.Sprintf("%s.%s", serviceName, namespace))             // service.namespace
		domains = append(domains, fmt.Sprintf("%s.%s.svc", serviceName, namespace))         // service.namespace.svc
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster", serviceName, namespace)) // service.namespace.svc.cluster
		// Always add the name of the service. This can be local, global, or the remote specific remote cluster.
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)) // service.namespace.svc.cluster.local
	}

	cp := make([]string, len(domains))
	copy(cp, domains)

	// Only used to get the ports...
	ports, err := f.GetPortToProtocolMappingForService(svc)
	if err != nil {
		log.Err(err).Msgf("Error getting ports for service %s", svc)
	}
	for _, domain := range cp {
		for port := range ports {
			domains = append(domains, fmt.Sprintf("%s:%d", domain, port)) // Add the port
		}
	}
	return domains
}

func (f fakeClient) GetTargetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	return map[uint32]string{uint32(tests.Endpoint.Port): "http"}, nil
}

func (f fakeClient) GetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	return map[uint32]string{uint32(tests.Endpoint.Port): "http"}, nil
}

// GetID returns the unique identifier of the Provider.
func (f fakeClient) GetID() string {
	return "Fake Kubernetes Client"
}

func (f fakeClient) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	endpoints, found := f.endpoints[svc.String()]
	if !found {
		return nil, errServiceNotFound
	}
	return endpoints, nil
}

func (f fakeClient) GetServicesByNameNamespace(name, namespace string) (ret []service.MeshService) {
	for _, services := range f.services {
		for _, svc := range services {
			if svc.Name == name && svc.Namespace == namespace {
				ret = append(ret, svc)
			}
		}
	}
	return
}

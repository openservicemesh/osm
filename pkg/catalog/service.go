package catalog

import (
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/utils"
)

// isTrafficSplitBackendService returns true if the given service is a backend service in any traffic split
func (mc *MeshCatalog) isTrafficSplitBackendService(svc service.MeshService) bool {
	for _, split := range mc.meshSpec.ListTrafficSplits() {
		for _, backend := range split.Spec.Backends {
			backendService := service.MeshService{
				Name:      backend.Service,
				Namespace: split.ObjectMeta.Namespace,
			}
			if svc.Equals(backendService) {
				return true
			}
		}
	}
	return false
}

// isTrafficSplitApexService returns true if the given service is an apex service in any traffic split
func (mc *MeshCatalog) isTrafficSplitApexService(svc service.MeshService) bool {
	for _, split := range mc.meshSpec.ListTrafficSplits() {
		apexService := service.MeshService{
			Name:      k8s.GetServiceFromHostname(split.Spec.Service),
			Namespace: split.Namespace,
		}
		if svc.Equals(apexService) {
			return true
		}
	}
	return false
}

// getApexServicesForBackendService returns a list of services that serve as the apex service in a traffic split where the
// given service is a backend
func (mc *MeshCatalog) getApexServicesForBackendService(targetService service.MeshService) []service.MeshService {
	var apexList []service.MeshService
	apexSet := mapset.NewSet()
	for _, split := range mc.meshSpec.ListTrafficSplits() {
		for _, backend := range split.Spec.Backends {
			if backend.Service == targetService.Name && split.Namespace == targetService.Namespace {
				meshService := service.MeshService{
					Name:      k8s.GetServiceFromHostname(split.Spec.Service),
					Namespace: split.Namespace,
				}
				apexSet.Add(meshService)
				break
			}
		}
	}

	for v := range apexSet.Iter() {
		apexList = append(apexList, v.(service.MeshService))
	}

	return apexList
}

// getServicesForServiceIdentity returns a list of services corresponding to a service identity
func (mc *MeshCatalog) getServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) ([]service.MeshService, error) {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		providerServices, err := provider.GetServicesForServiceIdentity(svcIdentity)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting K8s Services linked to Service Identity %s from provider %s", svcIdentity, provider.GetID())
			continue
		}
		var svcs []string
		for _, svc := range providerServices {
			svcs = append(svcs, svc.String())
		}

		log.Trace().Msgf("Found K8s Services %s linked to Service Identity %s from provider %s", strings.Join(svcs, ","), svcIdentity, provider.GetID())
		services = append(services, providerServices...)
	}

	if len(services) == 0 {
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (mc *MeshCatalog) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	// Currently OSM uses kubernetes service accounts as service identities
	var serviceIdentities []identity.ServiceIdentity
	for _, provider := range mc.serviceProviders {
		serviceIDs, err := provider.ListServiceIdentitiesForService(svc)
		if err != nil {
			log.Err(err).Msgf("Error getting ServiceIdentities for Service %s", svc)
			return nil, err
		}

		serviceIdentities = append(serviceIdentities, serviceIDs...)
	}

	return serviceIdentities, nil
}

// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
// The ports returned are the actual ports on which the application exposes the service derived from the service's endpoints,
// ie. 'spec.ports[].targetPort' instead of 'spec.ports[].port' for a Kubernetes service.
// The function ensures the port:protocol mapping is the same across different endpoint providers for the service, and returns
// an error otherwise.
func (mc *MeshCatalog) GetTargetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	var portToProtocolMap, previous map[uint32]string

	for _, provider := range mc.serviceProviders {
		current, err := provider.GetTargetPortToProtocolMappingForService(svc)
		if err != nil {
			return nil, err
		}

		if previous != nil && !reflect.DeepEqual(previous, current) {
			log.Error().Msgf("Service %s does not have the same port:protocol map across providers: expected=%v, got=%v", svc, previous, current)
			return nil, errors.Errorf("Service %s does not have the same port:protocol mapping across providers", svc)
		}
		previous = current
	}
	portToProtocolMap = previous
	if portToProtocolMap == nil {
		return nil, errors.Errorf("Error fetching port:protocol mapping for service %s", svc)
	}

	return portToProtocolMap, nil
}

// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
func (mc *MeshCatalog) GetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	portToProtocolMap := make(map[uint32]string)

	for _, provider := range mc.serviceProviders {
		currentPortToProtocolMap, err := provider.GetPortToProtocolMappingForService(svc)
		if err != nil {
			return nil, err
		}
		for key, value := range currentPortToProtocolMap {
			if v, ok := portToProtocolMap[key]; ok && v != value {
				return nil, errors.Errorf("Unexpected protocol %s for port %d, expected protocol %s, service: %s, provider: %s", value, key, v, svc, provider.GetID())
			}
			portToProtocolMap[key] = value
		}
	}

	if len(portToProtocolMap) == 0 {
		return nil, errors.Errorf("Error fetching port:protocol mapping for service %s", svc)
	}

	return portToProtocolMap, nil
}

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	var services []service.MeshService

	for _, provider := range mc.serviceProviders {
		svcs, err := provider.ListServices()
		if err != nil {
			log.Error().Err(err).Msgf("Error listing services for provider %s", provider.GetID())
			continue
		}

		services = append(services, svcs...)
	}

	return services
}

// GetServiceHostnames returns a list of hostnames corresponding to the service.
// If the service is in the same namespace, it returns the shorthand hostname for the service that does not
// include its namespace, ex: bookstore, bookstore:80
func (mc *MeshCatalog) GetServiceHostnames(meshService service.MeshService, locality service.Locality) ([]string, error) {
	svc := utils.K8sSvcToMeshSvc(mc.kubeController.GetService(meshService))

	var hostnames []string
	for _, provider := range mc.serviceProviders {
		hosts, err := provider.GetHostnamesForService(svc, locality)

		if err != nil {
			return nil, errors.Errorf("Error fetching hostnames for sercice: %s, provider: %s", meshService, provider.GetID())
		}
		hostnames = append(hostnames, hosts...)
	}

	return hostnames, nil
}

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}

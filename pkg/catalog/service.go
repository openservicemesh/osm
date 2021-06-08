package catalog

import (
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/kubernetes"
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
			Name:      kubernetes.GetServiceFromHostname(split.Spec.Service),
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
					Name:      kubernetes.GetServiceFromHostname(split.Spec.Service),
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

// getServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) getServicesForServiceAccount(sa identity.K8sServiceAccount) ([]service.MeshService, error) {
	var services []service.MeshService
	for _, provider := range mc.endpointsProviders {
		providerServices, err := provider.GetServicesForServiceAccount(sa)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting K8s Services linked to Service Account %s from provider %s", sa, provider.GetID())
			continue
		}
		var svcs []string
		for _, svc := range providerServices {
			svcs = append(svcs, svc.String())
		}

		log.Trace().Msgf("Found K8s Services %s linked to Service Account %s from endpoint provider %s", strings.Join(svcs, ","), sa, provider.GetID())
		services = append(services, providerServices...)
	}

	if len(services) == 0 {
		return nil, ErrServiceNotFoundForAnyProvider
	}

	return services, nil
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (mc *MeshCatalog) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	// Currently OSM uses kubernetes service accounts as service identities
	serviceAccounts, err := mc.kubeController.ListServiceIdentitiesForService(svc)
	if err != nil {
		log.Err(err).Msgf("Error getting ServiceAccounts for Service %s", svc)
		return nil, err
	}

	var serviceIdentities []identity.ServiceIdentity
	for _, svcAccount := range serviceAccounts {
		serviceIdentity := svcAccount.ToServiceIdentity()
		serviceIdentities = append(serviceIdentities, serviceIdentity)
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

	for _, provider := range mc.endpointsProviders {
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

	k8sSvc := mc.kubeController.GetService(svc)
	if k8sSvc == nil {
		return nil, errors.Wrapf(ErrServiceNotFound, "Error retrieving k8s service %s", svc)
	}

	for _, portSpec := range k8sSvc.Spec.Ports {
		var appProtocol string
		if portSpec.AppProtocol != nil {
			appProtocol = *portSpec.AppProtocol
		} else {
			appProtocol = kubernetes.GetAppProtocolFromPortName(portSpec.Name)
		}
		portToProtocolMap[uint32(portSpec.Port)] = appProtocol
	}

	return portToProtocolMap, nil
}

// listMeshServices returns all services in the mesh
func (mc *MeshCatalog) listMeshServices() []service.MeshService {
	var services []service.MeshService
	for _, svc := range mc.kubeController.ListServices() {
		services = append(services, utils.K8sSvcToMeshSvc(svc))
	}
	return services
}

// GetServiceHostnames returns a list of hostnames corresponding to the service.
// If the service is in the same namespace, it returns the shorthand hostname for the service that does not
// include its namespace, ex: bookstore, bookstore:80
func (mc *MeshCatalog) GetServiceHostnames(meshService service.MeshService, locality service.Locality) ([]string, error) {
	svc := mc.kubeController.GetService(meshService)
	if svc == nil {
		return nil, errors.Errorf("Error fetching service %q", meshService)
	}

	hostnames := kubernetes.GetHostnamesForService(svc, locality)
	return hostnames, nil
}

func getDefaultWeightedClusterForService(meshService service.MeshService) service.WeightedCluster {
	return service.WeightedCluster{
		ClusterName: service.ClusterName(meshService.String()),
		Weight:      constants.ClusterWeightAcceptAll,
	}
}

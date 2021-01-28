package catalog

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) GetServicesForServiceAccount(sa service.K8sServiceAccount) ([]service.MeshService, error) {
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
		return nil, errServiceNotFoundForAnyProvider
	}

	return services, nil
}

// ListServiceAccountsForService lists the service accounts associated with the given service
func (mc *MeshCatalog) ListServiceAccountsForService(svc service.MeshService) ([]service.K8sServiceAccount, error) {
	// Currently OSM uses kubernetes service accounts as service identities
	return mc.kubeController.ListServiceAccountsForService(svc)
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
		return nil, errors.Wrapf(errServiceNotFound, "Error retrieving k8s service %s", svc)
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

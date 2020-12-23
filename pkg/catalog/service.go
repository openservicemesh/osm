package catalog

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/service"
)

// GetServicesForServiceAccount returns a list of services corresponding to a service account
func (mc *MeshCatalog) GetServicesForServiceAccount(sa service.K8sServiceAccount) ([]service.MeshService, error) {
	var services []service.MeshService
	for _, provider := range mc.endpointsProviders {
		if providerServices, err := provider.GetServicesForServiceAccount(sa); err != nil {
			log.Warn().Msgf("Error getting K8s Services linked to Service Account %s from provider %s: %s", provider.GetID(), sa, err)
		} else {
			var svcs []string
			for _, svc := range providerServices {
				svcs = append(svcs, svc.String())
			}

			log.Trace().Msgf("Found K8s Services %s linked to Service Account %s from endpoint provider %s", strings.Join(svcs, ","), sa, provider.GetID())
			services = append(services, providerServices...)
		}
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

// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol.
// The function ensures the port:protocol mapping is the same across different endpoint providers for the service, and returns
// an error otherwise.
func (mc *MeshCatalog) GetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	var portToProtocolMap, previous map[uint32]string

	for _, provider := range mc.endpointsProviders {
		current, err := provider.GetPortToProtocolMappingForService(svc)
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

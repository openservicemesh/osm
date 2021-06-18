package kubernetes

import (
	networkingV1 "k8s.io/api/networking/v1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/discovery"

	"github.com/openservicemesh/osm/pkg/service"
)

const (
	// ingressKind denotes the Kind attribute of the Ingress k8s resource
	ingressKind = "Ingress"
)

var candidateVersions = []string{networkingV1.SchemeGroupVersion.String(), networkingV1beta1.SchemeGroupVersion.String()}

// GetIngressNetworkingV1beta1 returns the networking.k8s.io/v1beta1 ingress resources whose backends correspond to the service
func (c *Client) GetIngressNetworkingV1beta1(meshService service.MeshService) ([]*networkingV1beta1.Ingress, error) {
	var ingressResources []*networkingV1beta1.Ingress
	for _, ingressInterface := range c.informers[IngressesV1Beta1].GetStore().List() {
		ingress, ok := ingressInterface.(*networkingV1beta1.Ingress)
		if !ok {
			log.Error().Msg("Failed type assertion for Ingress in ingress cache")
			continue
		}

		// Extra safety - make sure we do not pay attention to Ingresses outside of observed namespaces
		if !c.IsMonitoredNamespace(ingress.Namespace) {
			continue
		}

		// Check if the ingress resource belongs to the same namespace as the service
		if ingress.Namespace != meshService.Namespace {
			// The ingress resource does not belong to the namespace of the service
			continue
		}
		if backend := ingress.Spec.Backend; backend != nil && backend.ServiceName == meshService.Name {
			// Default backend service
			ingressResources = append(ingressResources, ingress)
			continue
		}

	ingressRule:
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.ServiceName == meshService.Name {
					ingressResources = append(ingressResources, ingress)
					break ingressRule
				}
			}
		}
	}
	return ingressResources, nil
}

// GetIngressNetworkingV1 returns the networking.k8s.io/v1 ingress resources whose backends correspond to the service
func (c *Client) GetIngressNetworkingV1(meshService service.MeshService) ([]*networkingV1.Ingress, error) {
	var ingressResources []*networkingV1.Ingress
	for _, ingressInterface := range c.informers[IngressesV1].GetStore().List() {
		ingress, ok := ingressInterface.(*networkingV1.Ingress)
		if !ok {
			log.Error().Msg("Failed type assertion for Ingress in ingress cache")
			continue
		}

		// Extra safety - make sure we do not pay attention to Ingresses outside of observed namespaces
		if !c.IsMonitoredNamespace(ingress.Namespace) {
			continue
		}

		// Check if the ingress resource belongs to the same namespace as the service
		if ingress.Namespace != meshService.Namespace {
			// The ingress resource does not belong to the namespace of the service
			continue
		}
		if backend := ingress.Spec.DefaultBackend; backend != nil && backend.Service.Name == meshService.Name {
			// Default backend service
			ingressResources = append(ingressResources, ingress)
			continue
		}

	ingressRule:
		for _, rule := range ingress.Spec.Rules {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service.Name == meshService.Name {
					ingressResources = append(ingressResources, ingress)
					break ingressRule
				}
			}
		}
	}
	return ingressResources, nil
}

// getSupportedIngressVersions returns a map comprising of keys matching candidate ingress API versions
// and corresponding values indidicating if they are supported by the k8s API server or not. An error
// is returned in case this cannot be determined.
// Example return values:
// - only networking.k8s.io/v1 is supported: {'networking.k8s.io/v1': true, 'networking.k8s.io/v1beta1': false}, nil
// - only networking.k8s.io/v1beta1 is supported: {'networking.k8s.io/v1': false, 'networking.k8s.io/v1beta1': true}, nil
// - both networking.k8s.io/v1 and networking.k8s.io/v1beta1 are supported: {'networking.k8s.io/v1': true, 'networking.k8s.io/v1beta1': true}, nil
// - on error: nil, error
func getSupportedIngressVersions(client discovery.ServerResourcesInterface) (map[string]bool, error) {
	versions := make(map[string]bool)

	for _, groupVersion := range candidateVersions {
		// Initialize version support to false before checking with the k8s API server
		versions[groupVersion] = false

		list, err := client.ServerResourcesForGroupVersion(groupVersion)
		if err != nil {
			return nil, err
		}

		for _, elem := range list.APIResources {
			if elem.Kind == ingressKind {
				versions[groupVersion] = true
				break
			}
		}
	}

	return versions, nil
}

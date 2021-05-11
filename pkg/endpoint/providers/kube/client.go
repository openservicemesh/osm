package kube

import (
	"net"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, kubeController k8s.Controller, providerIdent string, cfg configurator.Configurator) (endpoint.Provider, error) {
	client := Client{
		providerIdent:  providerIdent,
		kubeClient:     kubeClient,
		kubeController: kubeController,
	}

	return &client, nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c Client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Trace().Msgf("[%s] Getting Endpoints for service %s on Kubernetes", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint

	kubernetesEndpoints, err := c.kubeController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Error().Err(err).Msgf("[%s] Error fetching Kubernetes Endpoints from cache for service %s", c.providerIdent, svc)
		return endpoints
	}

	if !c.kubeController.IsMonitoredNamespace(kubernetesEndpoints.Namespace) {
		// Doesn't belong to namespaces we are observing
		return endpoints
	}

	for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
		for _, address := range kubernetesEndpoint.Addresses {
			for _, port := range kubernetesEndpoint.Ports {
				ip := net.ParseIP(address.IP)
				if ip == nil {
					log.Error().Msgf("[%s] Error parsing IP address %s", c.providerIdent, address.IP)
					break
				}
				ept := endpoint.Endpoint{
					IP:   ip,
					Port: endpoint.Port(port.Port),
				}
				endpoints = append(endpoints, ept)
			}
		}
	}
	return endpoints
}

// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (c Client) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	sa := serviceIdentity.ToK8sServiceAccount()
	log.Trace().Msgf("[%s] Getting Endpoints for service account %s on Kubernetes", c.providerIdent, sa)
	var endpoints []endpoint.Endpoint

	podList := c.kubeController.ListPods()
	for _, pod := range podList {
		if pod.Namespace != sa.Namespace {
			continue
		}
		if pod.Spec.ServiceAccountName != sa.Name {
			continue
		}

		for _, podIP := range pod.Status.PodIPs {
			ip := net.ParseIP(podIP.IP)
			if ip == nil {
				log.Error().Msgf("[%s] Error parsing IP address %s", c.providerIdent, podIP.IP)
				break
			}
			ept := endpoint.Endpoint{IP: ip}
			endpoints = append(endpoints, ept)
		}
	}
	return endpoints
}

// GetServicesForServiceAccount retrieves a list of services for the given service account.
func (c Client) GetServicesForServiceAccount(svcAccount identity.K8sServiceAccount) ([]service.MeshService, error) {
	services := mapset.NewSet()

	for _, pod := range c.kubeController.ListPods() {
		if pod.Namespace != svcAccount.Namespace {
			continue
		}

		if pod.Spec.ServiceAccountName != svcAccount.Name {
			continue
		}

		podLabels := pod.ObjectMeta.Labels

		k8sServices, err := c.getServicesByLabels(podLabels, pod.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("[%s] Error retrieving service matching labels %v in namespace %s", c.providerIdent, podLabels, pod.Namespace)
			return nil, err
		}

		for _, svc := range k8sServices {
			services.Add(service.MeshService{
				Namespace: pod.Namespace,
				Name:      svc.Name,
			})
		}
	}

	if services.Cardinality() == 0 {
		log.Error().Err(errServiceNotFound).Msgf("[%s] No services for service account %s", c.providerIdent, svcAccount)
		return nil, errServiceNotFound
	}

	log.Trace().Msgf("[%s] Services for service account %s: %+v", c.providerIdent, svcAccount, services)
	servicesSlice := make([]service.MeshService, 0, services.Cardinality())
	for svc := range services.Iterator().C {
		servicesSlice = append(servicesSlice, svc.(service.MeshService))
	}

	return servicesSlice, nil
}

// GetTargetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol
func (c Client) GetTargetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	portToProtocolMap := make(map[uint32]string)

	endpoints, err := c.kubeController.GetEndpoints(svc)
	if err != nil || endpoints == nil {
		log.Error().Err(err).Msgf("[%s] Error fetching Kubernetes Endpoints from cache", c.providerIdent)
		return nil, err
	}

	if !c.kubeController.IsMonitoredNamespace(endpoints.Namespace) {
		return nil, errors.Errorf("Error fetching endpoints for service %s, namespace %s is not monitored", svc, endpoints.Namespace)
	}

	// A given port can only map to a single application protocol. Even if the same
	// port appears as a separate endpoint 'ip:port', the application protocol is
	// derived from the Service that fronts these endpoints, and a service's port
	// can only have one application protocol. So for the same port we don't have
	// to worry about different application protocols being set.
	for _, endpointSet := range endpoints.Subsets {
		for _, port := range endpointSet.Ports {
			var appProtocol string
			if port.AppProtocol != nil {
				appProtocol = *port.AppProtocol
			} else {
				appProtocol = k8s.GetAppProtocolFromPortName(port.Name)
				log.Debug().Msgf("endpoint port name: %s, appProtocol: %s", port.Name, appProtocol)
			}

			portToProtocolMap[uint32(port.Port)] = appProtocol
		}
	}

	return portToProtocolMap, nil
}

// getServicesByLabels gets Kubernetes services whose selectors match the given labels
func (c *Client) getServicesByLabels(podLabels map[string]string, namespace string) ([]corev1.Service, error) {
	var finalList []corev1.Service
	serviceList := c.kubeController.ListServices()

	for _, svc := range serviceList {
		// TODO: #1684 Introduce APIs to dynamically allow applying selectors, instead of callers implementing
		// filtering themselves
		if svc.Namespace != namespace {
			continue
		}

		svcRawSelector := svc.Spec.Selector
		// service has no selectors, we do not need to match against the pod label
		if len(svcRawSelector) == 0 {
			continue
		}
		selector := labels.Set(svcRawSelector).AsSelector()
		if selector.Matches(labels.Set(podLabels)) {
			finalList = append(finalList, *svc)
		}
	}

	return finalList, nil
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *Client) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	var err error

	// Check if the service has been given Cluster IP
	kubeService := c.kubeController.GetService(svc)
	if kubeService == nil {
		log.Error().Msgf("[%s] Could not find service %s", c.providerIdent, svc)
		return nil, errServiceNotFound
	}

	if len(kubeService.Spec.ClusterIP) == 0 || kubeService.Spec.ClusterIP == corev1.ClusterIPNone {
		// If service has no cluster IP or cluster IP is <none>, use final endpoint as resolvable destinations
		return c.ListEndpointsForService(svc), nil
	}

	// Cluster IP is present
	ip := net.ParseIP(kubeService.Spec.ClusterIP)
	if ip == nil {
		log.Error().Msgf("[%s] Could not parse Cluster IP %s", c.providerIdent, kubeService.Spec.ClusterIP)
		return nil, errParseClusterIP
	}

	for _, svcPort := range kubeService.Spec.Ports {
		endpoints = append(endpoints, endpoint.Endpoint{
			IP:   ip,
			Port: endpoint.Port(svcPort.Port),
		})
	}

	return endpoints, err
}

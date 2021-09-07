package kube

import (
	"net"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewClient returns a client that has all components necessary to connect to and maintain state of a Kubernetes cluster.
func NewClient(kubeController k8s.Controller, configClient config.Controller, providerIdent string, cfg configurator.Configurator) *client { //nolint:golint // exported func returns unexported type
	return &client{
		providerIdent:    providerIdent,
		kubeController:   kubeController,
		configClient:     configClient,
		meshConfigurator: cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *client) GetID() string {
	return c.providerIdent
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Trace().Msgf("[%s] Getting Endpoints for service %s on Kubernetes", c.providerIdent, svc)

	kubernetesEndpoints, err := c.kubeController.GetEndpoints(svc)
	if err != nil || kubernetesEndpoints == nil {
		log.Error().Err(err).Msgf("[%s] Error fetching Kubernetes Endpoints from cache for service %s", c.providerIdent, svc)
		return nil
	}

	if !c.kubeController.IsMonitoredNamespace(kubernetesEndpoints.Namespace) {
		// Doesn't belong to namespaces we are observing
		return nil
	}

	var endpoints []endpoint.Endpoint
	for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
		for _, port := range kubernetesEndpoint.Ports {
			// If a TargetPort is specified for the service, filter the endpoint by this port.
			// This is required to ensure we do not attempt to filter the endpoints when the endpoints
			// are being listed for a MeshService whose TargetPort is not known.
			if svc.TargetPort != 0 && port.Port != int32(svc.TargetPort) {
				// k8s service's port does not match MeshService port, ignore this port
				continue
			}
			for _, address := range kubernetesEndpoint.Addresses {
				ip := net.ParseIP(address.IP)
				if ip == nil {
					log.Error().Msgf("[%s] Error parsing endopint IP address %s for service %s", c.providerIdent, address.IP, svc)
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

	// Add multicluster service endpoints
	if c.meshConfigurator.GetFeatureFlags().EnableMulticlusterMode {
		endpoints = append(endpoints, c.getMulticlusterEndpoints(svc)...)
	}

	log.Trace().Msgf("[%s][ListEndpointsForService] Endpoints for service %s: %+v", c.providerIdent, svc, endpoints)

	return endpoints
}

// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (c *client) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
	sa := serviceIdentity.ToK8sServiceAccount()
	log.Trace().Msgf("[%s] (ListEndpointsForIdentity) Getting Endpoints for service account %s on Kubernetes", c.providerIdent, sa)

	var endpoints []endpoint.Endpoint
	for _, pod := range c.kubeController.ListPods() {
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

	// Add multicluster service endpoints
	if c.meshConfigurator.GetFeatureFlags().EnableMulticlusterMode {
		endpoints = append(endpoints, c.getMultiClusterServiceEndpointsForServiceAccount(sa.Name, sa.Namespace)...)
	}

	log.Trace().Msgf("[%s][ListEndpointsForIdentity] Endpoints for service identity (serviceAccount=%s) %s: %+v", c.providerIdent, serviceIdentity, sa, endpoints)

	return endpoints
}

// GetServicesForServiceIdentity retrieves a list of services for the given service identity.
func (c *client) GetServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) ([]service.MeshService, error) {
	var meshServices []service.MeshService
	svcSet := mapset.NewSet() // mapset is used to avoid duplicate elements in the output list

	svcAccount := svcIdentity.ToK8sServiceAccount()

	for _, pod := range c.kubeController.ListPods() {
		if pod.Namespace != svcAccount.Namespace {
			continue
		}

		if pod.Spec.ServiceAccountName != svcAccount.Name {
			continue
		}

		podLabels := pod.ObjectMeta.Labels
		meshServicesForPod, err := c.getServicesByLabels(podLabels, pod.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("[%s] Error retrieving service matching labels %v in namespace %s", c.providerIdent, podLabels, pod.Namespace)
			return nil, err
		}

		for _, svc := range meshServicesForPod {
			if added := svcSet.Add(svc); added {
				meshServices = append(meshServices, svc)
			}
		}
	}

	if len(meshServices) == 0 {
		log.Error().Err(errServiceNotFound).Msgf("[%s] No services for service account %s", c.providerIdent, svcAccount)
		return nil, errServiceNotFound
	}

	log.Trace().Msgf("[%s] Services for service account %s: %v", c.providerIdent, svcAccount, meshServices)
	return meshServices, nil
}

// getServicesByLabels gets Kubernetes services whose selectors match the given labels
// TODO(shashank): simplify method signature
func (c *client) getServicesByLabels(podLabels map[string]string, targetNamespace string) ([]service.MeshService, error) {
	var finalList []service.MeshService
	serviceList := c.kubeController.ListServices()

	for _, svc := range serviceList {
		// TODO: #1684 Introduce APIs to dynamically allow applying selectors, instead of callers implementing
		// filtering themselves
		if svc.Namespace != targetNamespace {
			continue
		}

		svcRawSelector := svc.Spec.Selector
		// service has no selectors, we do not need to match against the pod label
		if len(svcRawSelector) == 0 {
			continue
		}
		selector := labels.Set(svcRawSelector).AsSelector()
		if selector.Matches(labels.Set(podLabels)) {
			finalList = append(finalList, c.kubeController.K8sServiceToMeshServices(*svc)...)
		}
	}

	return finalList, nil
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved
func (c *client) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
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

// ListServices returns a list of services that are part of monitored namespaces
func (c *client) ListServices() ([]service.MeshService, error) {
	var services []service.MeshService
	for _, svc := range c.kubeController.ListServices() {
		services = append(services, c.kubeController.K8sServiceToMeshServices(*svc)...)
	}
	return services, nil
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (c *client) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	serviceAccounts, err := c.kubeController.ListServiceIdentitiesForService(svc)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ServiceAccounts for Service %s", svc)
		return nil, err
	}

	var serviceIdentities []identity.ServiceIdentity
	for _, svcAccount := range serviceAccounts {
		serviceIdentity := svcAccount.ToServiceIdentity()
		serviceIdentities = append(serviceIdentities, serviceIdentity)
	}

	return serviceIdentities, nil
}

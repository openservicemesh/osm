package kube

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/config"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/service"
	"github.com/openservicemesh/osm/pkg/utils"
)

// NewClient returns a client that has all components necessary to connect to and maintain state of a Kubernetes cluster.
func NewClient(kubeController k8s.Controller, configClient config.Controller, providerIdent string, cfg configurator.Configurator) *Client {
	return &Client{
		providerIdent:  providerIdent,
		kubeController: kubeController,
		configClient:   configClient,
		configurator:   cfg,
	}
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interfaces: EndpointsProvider, ServiceProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c *Client) ListEndpointsForService(svc service.MeshService) []endpoint.Endpoint {
	log.Trace().Msgf("[%s] Getting Endpoints for service %s on Kubernetes", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint
	var mcs *v1alpha1.MultiClusterService
	if !c.kubeController.IsMonitoredNamespace(svc.Namespace) {
		return nil
	}

	if svc.Local() {
		kubernetesEndpoints, err := c.kubeController.GetEndpoints(svc)
		if err != nil || kubernetesEndpoints == nil {
			log.Error().Err(err).Msgf("[%s] Error fetching Kubernetes Endpoints from cache for service %s", c.providerIdent, svc)
		} else {
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
		}
		return endpoints
	}
	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		mcs = c.configClient.GetMultiClusterService(svc.Name, svc.Namespace)
		if mcs == nil {
			return endpoints
		}

		if svc.SingleRemoteCluster() {
			for _, cluster := range mcs.Spec.Cluster {
				if cluster.Name == svc.ClusterDomain.String() {
					ep, err := epFromCluster(cluster.Address)
					if err == nil {
						return []endpoint.Endpoint{ep}
					}
				}
			}
			// Explicitly return nil for single remote cluster that couldn't get the cluster's address.
			return nil
		}

		if svc.Global() {
			for _, cluster := range mcs.Spec.Cluster {
				ep, err := epFromCluster(cluster.Address)
				if err != nil {
					log.Error().Err(err).Msgf("error parsing endpoint from %s", cluster.Address)
					continue
				}
				endpoints = append(endpoints, ep)
			}
		}
	}

	// Return the collected set of endpoints.
	return endpoints
}

func epFromCluster(addr string) (endpoint.Endpoint, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return endpoint.Endpoint{}, fmt.Errorf("expected an ip Address of format ip:port, got %s", addr)
	}
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return endpoint.Endpoint{}, fmt.Errorf("error parsing ip from %s, %v", addr, parts)
	}

	port, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return endpoint.Endpoint{}, err
	}

	return endpoint.Endpoint{
		IP:   ip,
		Port: endpoint.Port(port),
	}, nil
}

// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
// Note: ServiceIdentity must be in the format "name.namespace" [https://github.com/openservicemesh/osm/issues/3188]
func (c *Client) ListEndpointsForIdentity(serviceIdentity identity.ServiceIdentity) []endpoint.Endpoint {
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
	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		for _, mcs := range c.configClient.ListMultiClusterServices() {
			if mcs.Namespace != sa.Namespace {
				continue
			}
			if mcs.Spec.ServiceAccount != sa.Name {
				continue
			}
			for _, cluster := range mcs.Spec.Cluster {
				ep, err := epFromCluster(cluster.Address)
				if err != nil {
					log.Error().Err(err).Msgf("[%s] Error parsing IP address from MultiClusterService %s", c.providerIdent, cluster.Address)
					continue
				}
				endpoints = append(endpoints, ep)
			}
		}
	}
	return endpoints
}

// GetServicesForServiceIdentity retrieves a list of services for the given service identity.
func (c *Client) GetServicesForServiceIdentity(svcIdentity identity.ServiceIdentity) ([]service.MeshService, error) {
	services := mapset.NewSet()

	svcAccount := svcIdentity.ToK8sServiceAccount()

	for _, pod := range c.kubeController.ListPods() {
		if pod.Namespace != svcAccount.Namespace {
			continue
		}

		if pod.Spec.ServiceAccountName != svcAccount.Name {
			continue
		}

		podLabels := pod.ObjectMeta.Labels

		for _, svc := range c.getServicesByLabels(podLabels, pod.Namespace) {
			services.Add(service.MeshService{
				Namespace:     pod.Namespace,
				Name:          svc.Name,
				ClusterDomain: constants.LocalDomain,
			})
		}
	}

	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		mcservices := c.configClient.ListMultiClusterServices()
		for _, mcs := range mcservices {
			if mcs.Namespace != svcAccount.Namespace {
				continue
			}
			if mcs.Spec.ServiceAccount != svcAccount.Name {
				continue
			}
			for _, cluster := range mcs.Spec.Cluster {
				services.Add(service.MeshService{
					Namespace:     mcs.Namespace,
					Name:          mcs.Name,
					ClusterDomain: constants.ClusterDomain(cluster.Name),
				})
			}
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
func (c *Client) GetTargetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	if !svc.Local() {
		return nil, fmt.Errorf("cannot get target ports for remote service %s", svc)
	}
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
func (c *Client) getServicesByLabels(podLabels map[string]string, namespace string) []service.MeshService {
	var finalList []service.MeshService
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
			finalList = append(finalList, utils.K8sSvcToMeshSvc(svc))
		}
	}

	return finalList
}

// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service
// FQDN is resolved. This is used after DNS lookup in the filterchain match to match the IP address to an upstream
// service.
func (c *Client) GetResolvableEndpointsForService(svc service.MeshService) ([]endpoint.Endpoint, error) {
	var endpoints []endpoint.Endpoint
	var err error

	if svc.Local() {
		// Check if the service has been given Cluster IP
		kubeService := c.kubeController.GetService(svc)
		if kubeService == nil {
			log.Error().Msgf("[%s] Could not find service %s", c.providerIdent, svc)
			return nil, errServiceNotFound
		}

		// TODO(steeling): This is not how statefulsets work within kubernetes, and should be removed.
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

	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		mcs := c.configClient.GetMultiClusterService(svc.Name, svc.Namespace)
		if mcs == nil {
			return nil, errServiceNotFound
		}

		var ipStr string
		if svc.Global() {
			ipStr = mcs.Spec.GlobalIP
		} else {
			// Refers to a specific cluster
			for _, cluster := range mcs.Spec.Cluster {
				if cluster.Name == svc.ClusterDomain.String() {
					parts := strings.Split(cluster.Address, ":")
					if len(parts) != 2 {
						return nil, fmt.Errorf("expected address in format ip:port, got %s", cluster.Address)
					}
					ipStr = parts[0]
					break
				}
			}
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			log.Error().Msgf("[%s] Could not parse Cluster IP %s for service %s", c.providerIdent, ipStr, svc)
			return nil, errParseClusterIP
		}

		for _, port := range mcs.Spec.Ports {
			endpoints = append(endpoints, endpoint.Endpoint{
				IP:   ip,
				Port: endpoint.Port(port.Port),
			})
		}
	}

	return endpoints, nil
}

// ListServices returns a list of services that are part of monitored namespaces
func (c *Client) ListServices() ([]service.MeshService, error) {
	var services []service.MeshService
	for _, svc := range c.kubeController.ListServices() {
		services = append(services, utils.K8sSvcToMeshSvc(svc))
	}

	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		for _, mcs := range c.configClient.ListMultiClusterServices() {
			for _, cluster := range mcs.Spec.Cluster {
				services = append(services, service.MeshService{
					Namespace:     mcs.Namespace,
					Name:          mcs.Name,
					ClusterDomain: constants.ClusterDomain(cluster.Name),
				})
			}
		}
	}

	return services, nil
}

// ListServiceIdentitiesForService lists the service identities associated with the given mesh service.
func (c *Client) ListServiceIdentitiesForService(svc service.MeshService) ([]identity.ServiceIdentity, error) {
	if svc.Local() {
		serviceAccounts, err := c.kubeController.ListServiceAccountsForService(svc)
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

	mcs := c.configClient.GetMultiClusterService(svc.Name, svc.Namespace)
	if mcs == nil {
		err := fmt.Errorf("Error getting ServiceAccounts for Service %s", svc)
		log.Err(err)
		return nil, err
	}
	return []identity.ServiceIdentity{
		identity.K8sServiceAccount{Name: mcs.Spec.ServiceAccount, Namespace: mcs.Namespace}.ToServiceIdentity(),
	}, nil
}

// GetPortToProtocolMappingForService returns a mapping of the service's ports to their corresponding application protocol,
// where the ports returned are the ones used by downstream clients in their requests. This can be different from the ports
// actually exposed by the application binary, ie. 'spec.ports[].port' instead of 'spec.ports[].targetPort' for a Kubernetes service.
func (c *Client) GetPortToProtocolMappingForService(svc service.MeshService) (map[uint32]string, error) {
	portToProtocolMap := make(map[uint32]string)

	if svc.Local() {
		k8sSvc := c.kubeController.GetService(svc)
		if k8sSvc == nil {
			return nil, errors.Wrapf(errServiceNotFound, "Error retrieving k8s service %s", svc)
		}

		for _, portSpec := range k8sSvc.Spec.Ports {
			var appProtocol string
			if portSpec.AppProtocol != nil {
				appProtocol = *portSpec.AppProtocol
			} else {
				appProtocol = k8s.GetAppProtocolFromPortName(portSpec.Name)
			}
			portToProtocolMap[uint32(portSpec.Port)] = appProtocol
		}
		return portToProtocolMap, nil
	}

	if c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		mcs := c.configClient.GetMultiClusterService(svc.Name, svc.Namespace)
		if mcs == nil {
			return nil, fmt.Errorf("Error getting MultiClusterService for Service %s", svc)
		}

		for _, port := range mcs.Spec.Ports {
			if protocol, ok := portToProtocolMap[port.Port]; ok && protocol != port.Protocol {
				log.Error().Msgf("received conflicting port to protocol mapping for service %s", svc)
			}
			portToProtocolMap[port.Port] = port.Protocol
		}
	}
	return portToProtocolMap, nil
}

// GetHostnamesForService returns a list of hostnames over which the service can be accessed within the local cluster.
func (c *Client) GetHostnamesForService(svc service.MeshService, locality service.Locality) []string {
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
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster.%s", serviceName, namespace, constants.LocalDomain)) // service.namespace.svc.cluster.local
	}

	// Allow the local cluster name for all local services
	if svc.Local() && c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		fmt.Println("||||||||||||| allowed multicluster mode")
		// Allow reference to the self cluster id.
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster.%s", serviceName, namespace, c.configurator.GetClusterDomain()))
	}

	if !svc.Local() && c.configurator.GetFeatureFlags().EnableMulticlusterMode {
		// Always add the name of the service for remote services. This can be local, global, or the specific remote cluster.
		domains = append(domains, fmt.Sprintf("%s.%s.svc.cluster.%s", serviceName, namespace, svc.ClusterDomain.String())) // service.namespace.svc.cluster.local
	}

	cp := make([]string, len(domains))
	copy(cp, domains)

	// Only used to get the ports...
	ports, err := c.GetPortToProtocolMappingForService(svc)
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

package kube

import (
	"net"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, kubeController k8s.Controller, stop chan struct{}, providerIdent string, cfg configurator.Configurator) (endpoint.Provider, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
		Pods:      informerFactory.Core().V1().Pods().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints: informerCollection.Endpoints.GetStore(),
		Pods:      informerCollection.Pods.GetStore(),
	}

	client := Client{
		providerIdent:  providerIdent,
		kubeClient:     kubeClient,
		informers:      &informerCollection,
		caches:         &cacheCollection,
		cacheSynced:    make(chan interface{}),
		announcements:  make(chan a.Announcement),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}
	eptEventTypes := k8s.EventTypes{
		Add:    a.EndpointAdded,
		Update: a.EndpointUpdated,
		Delete: a.EndpointDeleted,
	}
	informerCollection.Endpoints.AddEventHandler(k8s.GetKubernetesEventHandlers("Endpoints", "Kubernetes", client.announcements, shouldObserve, nil, eptEventTypes))

	podEventTypes := k8s.EventTypes{
		Add:    a.PodAdded,
		Update: a.PodUpdated,
		Delete: a.PodDeleted,
	}
	informerCollection.Pods.AddEventHandler(k8s.GetKubernetesEventHandlers("Pods", "Kubernetes", client.announcements, shouldObserve, getPodUID, podEventTypes))

	if err := client.run(stop); err != nil {
		return nil, errors.Errorf("Failed to start Kubernetes EndpointProvider client: %+v", err)
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
	endpointsInterface, exist, err := c.caches.Endpoints.GetByKey(svc.String())
	if err != nil {
		log.Error().Err(err).Msgf("[%s] Error fetching Kubernetes Endpoints from cache", c.providerIdent)
		return endpoints
	}

	if !exist {
		log.Error().Msgf("[%s] Error fetching Kubernetes Endpoints from cache: MeshService %s does not exist", c.providerIdent, svc)
		return endpoints
	}

	kubernetesEndpoints := endpointsInterface.(*corev1.Endpoints)
	if kubernetesEndpoints != nil {
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
	}
	return endpoints
}

// GetServicesForServiceAccount retrieves a list of services for the given service account.
func (c Client) GetServicesForServiceAccount(svcAccount service.K8sServiceAccount) ([]service.MeshService, error) {
	services := mapset.NewSet()

	for _, podInterface := range c.caches.Pods.List() {
		pod := podInterface.(*corev1.Pod)
		if pod == nil {
			continue
		}

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
		// Add a service, which is a representation of the ServiceAccount, but not a real K8s service.
		// This will ensure that all pods in the service account are represented as one service.
		synthService := svcAccount.GetSyntheticService()
		services.Add(synthService)
		log.Trace().Msgf("[%s] No services for service account %s/%s; Adding synthetic service %s", c.providerIdent, svcAccount.Name, svcAccount.Namespace, synthService)
	} else {
		log.Trace().Msgf("[%s] Services for service account %s: %+v", c.providerIdent, svcAccount, services)
	}

	servicesSlice := make([]service.MeshService, 0, services.Cardinality())
	for svc := range services.Iterator().C {
		servicesSlice = append(servicesSlice, svc.(service.MeshService))
	}

	return servicesSlice, nil
}

// GetAnnouncementsChannel returns the announcement channel for the Kubernetes endpoints provider.
func (c Client) GetAnnouncementsChannel() <-chan a.Announcement {
	return c.announcements
}

func (c *Client) run(stop <-chan struct{}) error {
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[string]cache.SharedInformer{
		"Endpoints": c.informers.Endpoints,
		"Pods":      c.informers.Pods,
	}

	var names []string
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		log.Info().Msgf("[%s] Starting informer %s", c.providerIdent, name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("[%s] Waiting for informer's cache to sync: %+v", c.providerIdent, strings.Join(names, ", "))
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("[%s] Cache sync finished for %+v", c.providerIdent, names)
	return nil
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
		log.Error().Msgf("[%s] Could not find service %s", c.providerIdent, svc.String())
		return nil, errServiceNotFound
	}

	if len(kubeService.Spec.ClusterIP) == 0 {
		// If service has no cluster IP, use final endpoint as resolvable destinations
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

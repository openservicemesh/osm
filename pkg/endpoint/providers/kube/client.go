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
		announcements:  make(chan interface{}),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}
	informerCollection.Endpoints.AddEventHandler(k8s.GetKubernetesEventHandlers("Endpoints", "Kubernetes", client.announcements, shouldObserve))
	informerCollection.Pods.AddEventHandler(k8s.GetKubernetesEventHandlers("Pods", "Kubernetes", client.announcements, shouldObserve))

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
	log.Info().Msgf("[%s] Getting Endpoints for service %s on Kubernetes", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint = []endpoint.Endpoint{}
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
						log.Error().Msgf("Error parsing IP address %s", address.IP)
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
	log.Info().Msgf("[%s] Getting Services for service account %s on Kubernetes", c.providerIdent, svcAccount)
	services := mapset.NewSet()
	podsInterface := c.caches.Pods.List()

	for _, pods := range podsInterface {
		kubernetesPods := pods.(*corev1.Pod)
		if kubernetesPods == nil || !c.kubeController.IsMonitoredNamespace(kubernetesPods.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		spec := kubernetesPods.Spec
		namespacedSvcAccount := service.K8sServiceAccount{
			Namespace: kubernetesPods.Namespace,
			Name:      spec.ServiceAccountName,
		}
		if svcAccount != namespacedSvcAccount {
			continue
		}
		podLabels := kubernetesPods.ObjectMeta.Labels

		appNamspace := kubernetesPods.Namespace
		k8sServices, err := c.getServicesByLabels(podLabels, appNamspace)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving service matching labels %v in namespace %s", podLabels, appNamspace)
			return nil, errDidNotFindServiceForServiceAccount
		}
		for _, svc := range k8sServices {
			meshService := service.MeshService{
				Namespace: appNamspace,
				Name:      svc.Name,
			}
			services.Add(meshService)
		}
	}

	if services.Cardinality() == 0 {
		log.Error().Msgf("Did not find any service with serviceAccount = %s in namespace %s", svcAccount.Name, svcAccount.Namespace)

		return nil, errDidNotFindServiceForServiceAccount
	}

	log.Info().Msgf("[%s] Services %v observed on service account %s on Kubernetes", c.providerIdent, services, svcAccount)

	servicesSlice := make([]service.MeshService, 0, services.Cardinality())
	for svc := range services.Iterator().C {
		servicesSlice = append(servicesSlice, svc.(service.MeshService))
	}

	return servicesSlice, nil
}

// GetAnnouncementsChannel returns the announcement channel for the Kubernetes endpoints provider.
func (c Client) GetAnnouncementsChannel() <-chan interface{} {
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
		log.Debug().Msgf("Starting informer %s", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("Waiting for informer's cache to sync: %+v", strings.Join(names, ", "))
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for %+v", names)
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
	service := c.kubeController.GetService(svc)
	if service == nil {
		log.Error().Msgf("Could not find service %s", svc.String())
		return nil, errServiceNotFound
	}

	if len(service.Spec.ClusterIP) == 0 {
		// If service has no cluster IP, use final endpoint as resolvable destinations
		return c.ListEndpointsForService(svc), nil
	}

	// Cluster IP is present
	ip := net.ParseIP(service.Spec.ClusterIP)
	if ip == nil {
		log.Error().Msgf("Could not parse Cluster IP %s", service.Spec.ClusterIP)
		return nil, errParseClusterIP
	}

	for _, svcPort := range service.Spec.Ports {
		endpoints = append(endpoints, endpoint.Endpoint{
			IP:   ip,
			Port: endpoint.Port(svcPort.Port),
		})
	}

	return endpoints, err
}

package kube

import (
	"net"
	"reflect"
	"strings"

	mapset "github.com/deckarep/golang-set"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/namespace"
)

const namespaceSelectorLabel = "app"

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeClient kubernetes.Interface, namespaceController namespace.Controller, stop chan struct{}, providerIdent string, cfg configurator.Configurator) (*Client, error) {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		Endpoints:   informerFactory.Core().V1().Endpoints().Informer(),
		Deployments: informerFactory.Apps().V1().Deployments().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints:   informerCollection.Endpoints.GetStore(),
		Deployments: informerCollection.Deployments.GetStore(),
	}

	client := Client{
		providerIdent:       providerIdent,
		kubeClient:          kubeClient,
		informers:           &informerCollection,
		caches:              &cacheCollection,
		cacheSynced:         make(chan interface{}),
		announcements:       make(chan interface{}),
		namespaceController: namespaceController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return namespaceController.IsMonitoredNamespace(ns)
	}
	informerCollection.Endpoints.AddEventHandler(k8s.GetKubernetesEventHandlers("Endpoints", "Kubernetes", client.announcements, shouldObserve))
	informerCollection.Deployments.AddEventHandler(k8s.GetKubernetesEventHandlers("Deployments", "Kubernetes", client.announcements, shouldObserve))

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
		if !c.namespaceController.IsMonitoredNamespace(kubernetesEndpoints.Namespace) {
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

// GetServiceForServiceAccount retrieves the service for the given service account
func (c Client) GetServiceForServiceAccount(svcAccount service.K8sServiceAccount) (service.MeshService, error) {
	log.Info().Msgf("[%s] Getting Services for service account %s on Kubernetes", c.providerIdent, svcAccount)
	services := mapset.NewSet()
	deploymentsInterface := c.caches.Deployments.List()

	for _, deployments := range deploymentsInterface {
		kubernetesDeployments := deployments.(*appsv1.Deployment)
		if kubernetesDeployments != nil {
			if !c.namespaceController.IsMonitoredNamespace(kubernetesDeployments.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			spec := kubernetesDeployments.Spec
			namespacedSvcAccount := service.K8sServiceAccount{
				Namespace: kubernetesDeployments.Namespace,
				Name:      spec.Template.Spec.ServiceAccountName,
			}
			if svcAccount == namespacedSvcAccount {
				var selectorLabel map[string]string
				if spec.Selector != nil {
					selectorLabel = spec.Selector.MatchLabels
				} else {
					selectorLabel = spec.Template.Labels
				}
				namespacedService := service.MeshService{
					Namespace: kubernetesDeployments.Namespace,
					Name:      selectorLabel[namespaceSelectorLabel],
				}
				services.Add(namespacedService)
			}
		}
	}

	if services.Cardinality() == 0 {
		log.Error().Msgf("Did not find any service with serviceAccount = %s in namespace %s", svcAccount.Name, svcAccount.Namespace)
		return service.MeshService{}, errDidNotFindServiceForServiceAccount
	}

	// --- CONVENTION ---
	// By Open Service Mesh convention the number of services for a service account is 1
	// This is a limitation we set in place in order to make the mesh easy to understand and reason about.
	// When a service account has more than one service XDS will not apply any SMI policy for that service, leaving it out of the mesh.
	if services.Cardinality() > 1 {
		log.Error().Msgf("Found more than one service for serviceAccount %s in namespace %s; There should be only one!", svcAccount.Name, svcAccount.Namespace)
		return service.MeshService{}, errMoreThanServiceForServiceAccount
	}

	log.Info().Msgf("[%s] Services %v observed on service account %s on Kubernetes", c.providerIdent, services, svcAccount)
	svc := services.Pop().(service.MeshService)
	log.Trace().Msgf("Found service %s for serviceAccount %s in namespace %s", svc.Name, svcAccount.Name, svcAccount.Namespace)
	return svc, nil
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
		"Endpoints":   c.informers.Endpoints,
		"Deployments": c.informers.Deployments,
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

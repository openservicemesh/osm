package kube

import (
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	extensionsv1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

var resyncPeriod = 10 * time.Second

const namespaceSelectorLabel = "app"

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeConfig *rest.Config, namespaceController namespace.Controller, stop chan struct{}, providerIdent string) *Client {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)

	informerCollection := InformerCollection{
		Endpoints:   informerFactory.Core().V1().Endpoints().Informer(),
		Deployments: informerFactory.Extensions().V1beta1().Deployments().Informer(),
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

	informerCollection.Endpoints.AddEventHandler(client.getResourceEventHandlers("Endpoints"))
	informerCollection.Deployments.AddEventHandler(client.getResourceEventHandlers("Deployments"))

	if err := client.run(stop); err != nil {
		glog.Fatalf("[%s] Could not start Kubernetes EndpointProvider client: %s", packageName, err)
	}

	return &client
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// ListEndpointsForService retrieves the list of IP addresses for the given service
func (c Client) ListEndpointsForService(svc endpoint.ServiceName) []endpoint.Endpoint {
	glog.Infof("[%s][%s] Getting Endpoints for service %s on Kubernetes", packageName, c.providerIdent, svc)
	var endpoints []endpoint.Endpoint
	endpointsInterface, exist, err := c.caches.Endpoints.GetByKey(string(svc))
	if err != nil {
		glog.Errorf("[%s][%s] Error fetching Kubernetes Endpoints from cache: %s", packageName, c.providerIdent, err)
		return endpoints
	}

	if !exist {
		glog.Errorf("[%s][%s] Error fetching Kubernetes Endpoints from cache: ServiceName %s does not exist", packageName, c.providerIdent, svc)
		return endpoints
	}

	if kubernetesEndpoints := endpointsInterface.(*corev1.Endpoints); kubernetesEndpoints != nil {
		if !c.namespaceController.IsMonitoredNamespace(kubernetesEndpoints.Namespace) {
			// Doesn't belong to namespaces we are observing
			return endpoints
		}
		for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
			for _, address := range kubernetesEndpoint.Addresses {
				for _, port := range kubernetesEndpoint.Ports {
					ept := endpoint.Endpoint{
						IP:   net.IP(address.IP),
						Port: endpoint.Port(port.Port),
					}
					endpoints = append(endpoints, ept)
				}

			}
		}
	}
	return endpoints
}

// ListServicesForServiceAccount retrieves the list of Services for the given service account
func (c Client) ListServicesForServiceAccount(svcAccount endpoint.NamespacedServiceAccount) []endpoint.NamespacedService {
	glog.Infof("[%s][%s] Getting Services for service account %s on Kubernetes", packageName, c.providerIdent, svcAccount)
	var services []endpoint.NamespacedService
	deploymentsInterface := c.caches.Deployments.List()

	for _, deployments := range deploymentsInterface {
		if kubernetesDeployments := deployments.(*extensionsv1.Deployment); kubernetesDeployments != nil {
			if !c.namespaceController.IsMonitoredNamespace(kubernetesDeployments.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			spec := kubernetesDeployments.Spec
			namespacedSvcAccount := endpoint.NamespacedServiceAccount{
				Namespace:      kubernetesDeployments.Namespace,
				ServiceAccount: spec.Template.Spec.ServiceAccountName,
			}
			if svcAccount == namespacedSvcAccount {
				var selectorLabel map[string]string
				if spec.Selector != nil {
					selectorLabel = spec.Selector.MatchLabels
				} else {
					selectorLabel = spec.Template.Labels
				}
				namespacedService := endpoint.NamespacedService{
					Namespace: kubernetesDeployments.Namespace,
					Service:   selectorLabel[namespaceSelectorLabel],
				}
				services = append(services, namespacedService)
			}
		}
	}

	glog.Infof("[%s][%s] Services %v observed on service account %s on Kubernetes", packageName, c.providerIdent, services, svcAccount)
	return services
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
		glog.V(level.Debug).Infof("[%s] Starting informer %s", packageName, name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(level.Info).Infof("[%s] Waiting for informer's cache to sync: %+v", packageName, strings.Join(names, ", "))
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(level.Info).Infof("Cache sync finished for %+v", names)
	return nil
}

package kube

import (
	"net"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smc/pkg/logging"
)

var resyncPeriod = 1 * time.Second

// NewProvider implements mesh.EndpointsProvider, which creates a new Kubernetes cluster/compute provider.
func NewProvider(kubeConfig *rest.Config, namespaces []string, announcements chan interface{}, stop chan struct{}, providerIdent string) *Client {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	var options []informers.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)

	informerCollection := InformerCollection{
		Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints: informerCollection.Endpoints.GetStore(),
	}

	client := Client{
		providerIdent: providerIdent,
		kubeClient:    kubeClient,
		informers:     &informerCollection,
		caches:        &cacheCollection,
		announcements: announcements,
		cacheSynced:   make(chan interface{}),
	}

	h := handlers{client}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	informerCollection.Endpoints.AddEventHandler(resourceHandler)

	if err := client.run(stop); err != nil {
		glog.Fatal("Could not start Kubernetes EndpointProvider client", err)
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
	glog.Infof("[%s] Getting Endpoints for service %s on Kubernetes", c.providerIdent, svc)
	var endpoints []endpoint.Endpoint
	endpointsInterface, exist, err := c.caches.Endpoints.GetByKey(string(svc))
	if err != nil {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: %s", c.providerIdent, err)
		return endpoints
	}

	if !exist {
		glog.Errorf("[%s] Error fetching Kubernetes Endpoints from cache: ServiceName %s does not exist", c.providerIdent, svc)
		return endpoints
	}

	// TODO(draychev): get the port number from the service
	port := endpoint.Port(15003)

	if kubernetesEndpoints := endpointsInterface.(*corev1.Endpoints); kubernetesEndpoints != nil {
		for _, kubernetesEndpoint := range kubernetesEndpoints.Subsets {
			for _, address := range kubernetesEndpoint.Addresses {
				ept := endpoint.Endpoint{
					IP:   net.IP(address.IP),
					Port: port,
				}
				endpoints = append(endpoints, ept)
			}
		}
	}
	return endpoints
}

func (c Client) GetAnnouncementsChannel() <-chan interface{} {
	// return c.announcements
	// TODO(draychev): implement
	return make(chan interface{})
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	glog.V(log.LvlInfo).Infoln("Kubernetes Compute Provider started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"Endpoints": c.informers.Endpoints,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		glog.Info("Starting informer: ", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(log.LvlInfo).Infof("Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(log.LvlInfo).Infof("Cache sync finished for %+v", names)
	return nil
}

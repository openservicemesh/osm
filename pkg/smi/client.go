package smi

import (
	"fmt"
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	"github.com/eapache/channels"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var resyncPeriod = 1 * time.Second

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "Specification"

// NewSpecificationClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewSpecificationClient(kubeConfig *rest.Config, namespaces []string, announcement *channels.RingChannel, stop chan struct{}) MeshSpec {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiClientset := versioned.NewForConfigOrDie(kubeConfig)

	announcements := channels.NewRingChannel(1204)
	client := newSMIClient(kubeClient, smiClientset, namespaces, announcements, kubernetesClientName)
	err := client.run(stop)
	if err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return client
}

// run executes informer collection.
func (c *Client) run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("SMI Client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"TrafficSplit": c.informers.TrafficSplit,
		"Services":     c.informers.Services,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		glog.Info("Starting informer: ", name)
		go informer.Run(stopCh)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(1).Infof("[SMI Client] Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stopCh, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(1).Infof("[SMI Client] Cache sync finished for %+v", names)
	return nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(kubeClient *kubernetes.Clientset, smiClient *versioned.Clientset, namespaces []string, announcements *channels.RingChannel, providerIdent string) *Client {
	var options []informers.SharedInformerOption
	var smiOptions []smiExternalVersions.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
		smiOptions = append(smiOptions, smiExternalVersions.WithNamespace(namespace))
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	smiInformerFactory := smiExternalVersions.NewSharedInformerFactoryWithOptions(smiClient, resyncPeriod, smiOptions...)
	informerCollection := InformerCollection{
		Services:     informerFactory.Core().V1().Services().Informer(),
		TrafficSplit: smiInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
	}

	cacheCollection := CacheCollection{
		Services:     informerCollection.Services.GetStore(),
		TrafficSplit: informerCollection.TrafficSplit.GetStore(),
	}

	client := Client{
		providerIdent: providerIdent,
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

	informerCollection.Services.AddEventHandler(resourceHandler)
	informerCollection.TrafficSplit.AddEventHandler(resourceHandler)

	return &client
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	var trafficSplits []*v1alpha2.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListServices implements mesh.MeshSpec by returning the services observed from the given compute provider
func (c *Client) ListServices() []endpoint.ServiceName {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var services []endpoint.ServiceName
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, split.Spec.Service)
		services = append(services, endpoint.ServiceName(namespacedServiceName))
		for _, backend := range split.Spec.Backends {
			namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
			services = append(services, endpoint.ServiceName(namespacedServiceName))
		}
	}
	return services
}

// GetService retrieves the Kubernetes Services resource for the given ServiceName.
func (c *Client) GetService(svc endpoint.ServiceName) (service *v1.Service, exists bool, err error) {
	svcIf, exists, err := c.caches.Services.GetByKey(string(svc))
	if exists && err == nil {
		return svcIf.(*v1.Service), exists, err
	}
	return nil, exists, err
}

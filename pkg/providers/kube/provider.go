package kube

import (
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/deislabs/smc/pkg/mesh"
)

// GetIPs retrieves the list of IP addresses for the given service
func (kp KubernetesProvider) GetIPs(svc mesh.ServiceName) []mesh.IP {
	glog.Infof("[kubernetes] Getting IPs for service %s", svc)
	var ips []mesh.IP
	endpointsInterface, exist, err := kp.Caches.Endpoints.GetByKey(string(svc))
	if err != nil {
		glog.Error("Error fetching endpoints from store, error occurred ", err)
		return ips
	}

	if !exist {
		glog.Error("Error fetching endpoints from store! ServiceName does not exist: ", svc)
		return ips
	}

	if endpoints := endpointsInterface.(*v1.Endpoints); endpoints != nil {
		for _, endpoint := range endpoints.Subsets {
			for _, address := range endpoint.Addresses {
				ips = append(ips, mesh.IP(address.IP))
			}
		}
	}
	return ips
}

// NewProvider creates a provider based on a Kubernetes client instance.
func NewProvider(kubeConfig *rest.Config, smiClient *versioned.Clientset, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel) *KubernetesProvider {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	var options []informers.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
	}

	var smiOptions []smiExternalVersions.SharedInformerOption
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	smiInformerFactory := smiExternalVersions.NewSharedInformerFactoryWithOptions(smiClient, resyncPeriod, smiOptions...)

	informerCollection := InformerCollection{
		Endpoints:    informerFactory.Core().V1().Endpoints().Informer(),
		Services:     informerFactory.Core().V1().Services().Informer(),
		TrafficSplit: smiInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints:    informerCollection.Endpoints.GetStore(),
		Services:     informerCollection.Services.GetStore(),
		TrafficSplit: informerCollection.TrafficSplit.GetStore(),
	}

	context := &KubernetesProvider{
		kubeClient:   kubeClient,
		informers:    &informerCollection,
		Caches:       &cacheCollection,
		announceChan: announceChan,
		CacheSynced:  make(chan interface{}),
	}

	h := handlers{context}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	informerCollection.Endpoints.AddEventHandler(resourceHandler)

	return context
}

// Run executes informer collection.
func (kp *KubernetesProvider) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("k8s provider run started")
	var hasSynced []cache.InformerSynced

	if kp.informers == nil {
		return errInitInformers
	}

	sharedInformers := []cache.SharedInformer{
		kp.informers.Endpoints,
		kp.informers.Services,
		kp.informers.TrafficSplit,
	}

	for _, informer := range sharedInformers {
		go informer.Run(stopCh)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(1).Infoln("Waiting for initial cache sync")
	if !cache.WaitForCacheSync(stopCh, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(kp.CacheSynced)

	glog.V(1).Infoln("initial cache sync done")
	glog.V(1).Infoln("k8s provider run finished")
	return nil
}

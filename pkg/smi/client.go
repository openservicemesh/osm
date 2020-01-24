package smi

import (
	"time"

	smiTrafficTargetClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficTargetExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/eapache/channels"
	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var resyncPeriod = 1 * time.Second

// Run executes informer collection.
func (c *Client) Run(stopCh <-chan struct{}) error {
	glog.V(1).Infoln("SMI Client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"TrafficSplit":  c.informers.TrafficSplit,
		"Services":      c.informers.Services,
		"TrafficSpec":   c.informers.TrafficSpec,
		"TrafficTarget": c.informers.TrafficTarget,
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
func newSMIClient(kubeClient *kubernetes.Clientset, smiTrafficSplitClient *smiTrafficSplitClientVersion.Clientset, smiTrafficSpecClient *smiTrafficSpecClientVersion.Clientset, smiTrafficTargetClient *smiTrafficTargetClientVersion.Clientset, namespaces []string, announceChan *channels.RingChannel, providerIdent string) *Client {
	var options []informers.SharedInformerOption
	var smiTrafficSplitOptions []smiTrafficSplitExternalVersions.SharedInformerOption
	var smiTrafficSpecOptions []smiTrafficSpecExternalVersions.SharedInformerOption
	var smiTrafficTargetOptions []smiTrafficTargetExternalVersions.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
		smiTrafficSplitOptions = append(smiTrafficSplitOptions, smiTrafficSplitExternalVersions.WithNamespace(namespace))
		smiTrafficSpecOptions = append(smiTrafficSpecOptions, smiTrafficSpecExternalVersions.WithNamespace(namespace))
		smiTrafficTargetOptions = append(smiTrafficTargetOptions, smiTrafficTargetExternalVersions.WithNamespace(namespace))
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	smiTrafficSplitInformerFactory := smiTrafficSplitExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficSplitClient, resyncPeriod, smiTrafficSplitOptions...)
	smiTrafficSpecInformerFactory := smiTrafficSpecExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficSpecClient, resyncPeriod, smiTrafficSpecOptions...)
	smiTrafficTargetInformerFactory := smiTrafficTargetExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficTargetClient, resyncPeriod, smiTrafficTargetOptions...)

	//todo(snchh) : the TrafficSpec is only listening for HTTPRouteGroups, need to extend for TCPRouteGroup
	informerCollection := InformerCollection{
		Services:      informerFactory.Core().V1().Services().Informer(),
		TrafficSplit:  smiTrafficSplitInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		TrafficSpec:   smiTrafficSpecInformerFactory.Specs().V1alpha1().HTTPRouteGroups().Informer(),
		TrafficTarget: smiTrafficTargetInformerFactory.Access().V1alpha1().TrafficTargets().Informer(),
	}

	cacheCollection := CacheCollection{
		Services:      informerCollection.Services.GetStore(),
		TrafficSplit:  informerCollection.TrafficSplit.GetStore(),
		TrafficSpec:   informerCollection.TrafficSpec.GetStore(),
		TrafficTarget: informerCollection.TrafficTarget.GetStore(),
	}

	client := Client{
		providerIdent: providerIdent,
		informers:     &informerCollection,
		caches:        &cacheCollection,
		announceChan:  announceChan,
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
	informerCollection.TrafficSpec.AddEventHandler(resourceHandler)
	informerCollection.TrafficTarget.AddEventHandler(resourceHandler)

	return &client
}

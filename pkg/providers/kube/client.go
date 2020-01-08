package kube

import (
	"time"

	"github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/eapache/channels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	smcClient "github.com/deislabs/smc/pkg/smc_client/clientset/versioned"
	smcInformers "github.com/deislabs/smc/pkg/smc_client/informers/externalversions"
)

// NewClient creates a provider based on a Kubernetes client instance.
func NewClient(kubeClient *kubernetes.Clientset, smiClient *versioned.Clientset, azureResourceClient *smcClient.Clientset, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel, providerIdent string) *Client {
	var options []informers.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)

	informerCollection := InformerCollection{
		Endpoints: informerFactory.Core().V1().Endpoints().Informer(),
		Services:  informerFactory.Core().V1().Services().Informer(),
	}

	// Service Mesh Interface informers are optional
	var smiOptions []smiExternalVersions.SharedInformerOption
	var smiInformerFactory smiExternalVersions.SharedInformerFactory
	if smiClient != nil {
		smiInformerFactory = smiExternalVersions.NewSharedInformerFactoryWithOptions(smiClient, resyncPeriod, smiOptions...)
		informerCollection.TrafficSplit = smiInformerFactory.Split().V1alpha2().TrafficSplits().Informer()
	}

	// Azure Resource CRD informers are optional
	var azureResourceOptions []smcInformers.SharedInformerOption
	var azureResourceFactory smcInformers.SharedInformerFactory
	if azureResourceClient != nil {
		azureResourceFactory = smcInformers.NewSharedInformerFactoryWithOptions(azureResourceClient, resyncPeriod, azureResourceOptions...)
		informerCollection.AzureResource = azureResourceFactory.Smc().V1().AzureResources().Informer()
	}

	cacheCollection := CacheCollection{
		Endpoints: informerCollection.Endpoints.GetStore(),
		Services:  informerCollection.Services.GetStore(),
	}

	if informerCollection.TrafficSplit != nil {
		cacheCollection.TrafficSplit = informerCollection.TrafficSplit.GetStore()
	}

	if informerCollection.AzureResource != nil {
		cacheCollection.AzureResource = informerCollection.AzureResource.GetStore()
	}

	client := Client{
		providerIdent: providerIdent,
		kubeClient:    kubeClient,
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

	informerCollection.Endpoints.AddEventHandler(resourceHandler)

	return &client
}

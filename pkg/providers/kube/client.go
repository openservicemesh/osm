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
func NewClient(kubeClient *kubernetes.Clientset, smiClient *versioned.Clientset, azureResourceClient *smcClient.Clientset, namespaces []string, resyncPeriod time.Duration, announceChan *channels.RingChannel) *Client {
	var options []informers.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)

	var smiOptions []smiExternalVersions.SharedInformerOption
	smiInformerFactory := smiExternalVersions.NewSharedInformerFactoryWithOptions(smiClient, resyncPeriod, smiOptions...)

	var azureResourceOptions []smcInformers.SharedInformerOption
	azureResourceFactory := smcInformers.NewSharedInformerFactoryWithOptions(azureResourceClient, resyncPeriod, azureResourceOptions...)

	informerCollection := InformerCollection{
		Endpoints:     informerFactory.Core().V1().Endpoints().Informer(),
		Services:      informerFactory.Core().V1().Services().Informer(),
		TrafficSplit:  smiInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		AzureResource: azureResourceFactory.Smc().V1().AzureResources().Informer(),
	}

	cacheCollection := CacheCollection{
		Endpoints:     informerCollection.Endpoints.GetStore(),
		Services:      informerCollection.Services.GetStore(),
		TrafficSplit:  informerCollection.TrafficSplit.GetStore(),
		AzureResource: informerCollection.AzureResource.GetStore(),
	}

	context := &Client{
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

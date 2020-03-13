package azure

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	smc "github.com/open-service-mesh/osm/pkg/apis/azureresource/v1"
	"github.com/open-service-mesh/osm/pkg/log/level"
	smcClient "github.com/open-service-mesh/osm/pkg/smc_client/clientset/versioned"
	smcInformers "github.com/open-service-mesh/osm/pkg/smc_client/informers/externalversions"
)

const (
	kubernetesClientName = "AzureResourceClient"
)

var resyncPeriod = 10 * time.Second

// NewClient creates the Kubernetes client, which retrieves the AzureResource CRD and Services resources.
func NewClient(kubeConfig *rest.Config, namespaces []string, stop chan struct{}) *Client {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	azureResourceClient := smcClient.NewForConfigOrDie(kubeConfig)

	k8sClient := newClient(kubeClient, azureResourceClient, namespaces)
	if err := k8sClient.Run(stop); err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return k8sClient
}

// newClient creates a provider based on a Kubernetes client instance.
func newClient(kubeClient *kubernetes.Clientset, azureResourceClient *smcClient.Clientset, namespaces []string) *Client {
	var options []smcInformers.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, smcInformers.WithNamespace(namespace))
	}
	azureResourceFactory := smcInformers.NewSharedInformerFactoryWithOptions(azureResourceClient, resyncPeriod, options...)
	informerCollection := InformerCollection{
		AzureResource: azureResourceFactory.Smc().V1().AzureResources().Informer(),
	}

	cacheCollection := CacheCollection{
		AzureResource: informerCollection.AzureResource.GetStore(),
	}

	client := Client{
		providerIdent: kubernetesClientName,
		kubeClient:    kubeClient,
		informers:     &informerCollection,
		caches:        &cacheCollection,
		cacheSynced:   make(chan interface{}),
		announcements: make(chan interface{}),
	}

	h := handlers{client}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	informerCollection.AzureResource.AddEventHandler(resourceHandler)

	return &client
}

// Run executes informer collection.
func (c *Client) Run(stop <-chan struct{}) error {
	glog.V(level.Info).Infoln("Kubernetes Compute Provider started")
	var hasSynced []cache.InformerSynced

	glog.Info("Starting AzureResource informer")
	go c.informers.AzureResource.Run(stop)
	hasSynced = append(hasSynced, c.informers.AzureResource.HasSynced)

	glog.V(level.Info).Infof("Waiting AzureResource informer cache sync")
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(level.Info).Info("Cache sync for AzureResource finished")
	return nil
}

// ListAzureResources lists the AzureResource CRD resources.
func (c *Client) ListAzureResources() []*smc.AzureResource {
	var azureResources []*smc.AzureResource
	for _, azureResourceInterface := range c.caches.AzureResource.List() {
		azureResource := azureResourceInterface.(*smc.AzureResource)
		azureResources = append(azureResources, azureResource)
	}
	return azureResources
}

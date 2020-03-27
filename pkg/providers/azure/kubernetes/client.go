package azure

import (
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	osm "github.com/open-service-mesh/osm/pkg/apis/azureresource/v1"
	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"github.com/open-service-mesh/osm/pkg/namespace"
	osmClient "github.com/open-service-mesh/osm/pkg/osm_client/clientset/versioned"
	osmInformers "github.com/open-service-mesh/osm/pkg/osm_client/informers/externalversions"
)

const (
	kubernetesClientName = "AzureResourceClient"
)

var resyncPeriod = 10 * time.Second

// NewClient creates the Kubernetes client, which retrieves the AzureResource CRD and Services resources.
func NewClient(kubeConfig *rest.Config, namespaceController namespace.Controller, stop chan struct{}) *Client {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	azureResourceClient := osmClient.NewForConfigOrDie(kubeConfig)

	k8sClient := newClient(kubeClient, azureResourceClient, namespaceController)
	if err := k8sClient.Run(stop); err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return k8sClient
}

// newClient creates a provider based on a Kubernetes client instance.
func newClient(kubeClient *kubernetes.Clientset, azureResourceClient *osmClient.Clientset, namespaceController namespace.Controller) *Client {
	azureResourceFactory := osmInformers.NewSharedInformerFactory(azureResourceClient, resyncPeriod)
	informerCollection := InformerCollection{
		AzureResource: azureResourceFactory.Osm().V1().AzureResources().Informer(),
	}

	cacheCollection := CacheCollection{
		AzureResource: informerCollection.AzureResource.GetStore(),
	}

	client := Client{
		providerIdent:       kubernetesClientName,
		kubeClient:          kubeClient,
		informers:           &informerCollection,
		caches:              &cacheCollection,
		cacheSynced:         make(chan interface{}),
		announcements:       make(chan interface{}),
		namespaceController: namespaceController,
	}

	informerCollection.AzureResource.AddEventHandler(k8s.GetKubernetesEventHandlers("AzureResource", "Azure", client.announcements))

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
func (c *Client) ListAzureResources() []*osm.AzureResource {
	var azureResources []*osm.AzureResource
	for _, azureResourceInterface := range c.caches.AzureResource.List() {
		azureResource := azureResourceInterface.(*osm.AzureResource)
		if !c.namespaceController.IsMonitoredNamespace(azureResource.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		azureResources = append(azureResources, azureResource)
	}
	return azureResources
}

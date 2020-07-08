package azure

import (
	"reflect"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	osm "github.com/open-service-mesh/osm/pkg/apis/azureresource/v1"
	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"

	"github.com/open-service-mesh/osm/pkg/namespace"
	osmClient "github.com/open-service-mesh/osm/pkg/osm_client/clientset/versioned"
	osmInformers "github.com/open-service-mesh/osm/pkg/osm_client/informers/externalversions"
)

const (
	kubernetesClientName = "AzureResourceClient"
)

// NewClient creates the Kubernetes client, which retrieves the AzureResource CRD and Services resources.
func NewClient(kubeConfig *rest.Config, namespaceController namespace.Controller, stop chan struct{}) *Client {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	azureResourceClient := osmClient.NewForConfigOrDie(kubeConfig)

	k8sClient := newClient(kubeClient, azureResourceClient, namespaceController)
	if err := k8sClient.Run(stop); err != nil {
		log.Fatal().Err(err).Msgf("Could not start %s client", kubernetesClientName)
	}
	return k8sClient
}

// newClient creates a provider based on a Kubernetes client instance.
func newClient(kubeClient *kubernetes.Clientset, azureResourceClient *osmClient.Clientset, namespaceController namespace.Controller) *Client {
	azureResourceFactory := osmInformers.NewSharedInformerFactory(azureResourceClient, k8s.DefaultKubeEventResyncInterval)
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

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return namespaceController.IsMonitoredNamespace(ns)
	}
	informerCollection.AzureResource.AddEventHandler(k8s.GetKubernetesEventHandlers("AzureResource", "Azure", client.announcements, shouldObserve))

	return &client
}

// Run executes informer collection.
func (c *Client) Run(stop <-chan struct{}) error {
	log.Info().Msg("Kubernetes Compute Provider started")
	var hasSynced []cache.InformerSynced

	log.Info().Msg("Starting AzureResource informer")
	go c.informers.AzureResource.Run(stop)
	hasSynced = append(hasSynced, c.informers.AzureResource.HasSynced)

	log.Info().Msgf("Waiting AzureResource informer cache sync")
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msg("Cache sync for AzureResource finished")
	return nil
}

// ListAzureResources lists the AzureResource CRD resources.
func (c *Client) ListAzureResources() []*osm.AzureResource {
	var azureResources []*osm.AzureResource
	for _, azureResourceInterface := range c.caches.AzureResource.List() {
		azureResource, ok := azureResourceInterface.(*osm.AzureResource)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msg("Failed type assertion for AzureResource in cache")
			continue
		}
		if !c.namespaceController.IsMonitoredNamespace(azureResource.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		azureResources = append(azureResources, azureResource)
	}
	return azureResources
}

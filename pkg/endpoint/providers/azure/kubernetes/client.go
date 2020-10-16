package azure

import (
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	osm "github.com/openservicemesh/osm/pkg/apis/azureresource/v1"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"

	"github.com/openservicemesh/osm/pkg/configurator"
	osmClient "github.com/openservicemesh/osm/pkg/osm_client/clientset/versioned"
	osmInformers "github.com/openservicemesh/osm/pkg/osm_client/informers/externalversions"
)

const (
	kubernetesClientName = "AzureResourceClient"
)

// NewClient creates the Kubernetes client, which retrieves the AzureResource CRD and Services resources.
func NewClient(kubeClient kubernetes.Interface, azureResourceKubeConfig *rest.Config, kubeController k8s.Controller, stop chan struct{}, cfg configurator.Configurator) (*Client, error) {
	azureResourceClient := osmClient.NewForConfigOrDie(azureResourceKubeConfig)

	k8sClient := newClient(kubeClient, azureResourceClient, kubeController)
	if err := k8sClient.Run(stop); err != nil {
		return nil, errors.Errorf("Failed to start %s client: %+v", kubernetesClientName, err)
	}
	return k8sClient, nil
}

// newClient creates a provider based on a Kubernetes client instance.
func newClient(kubeClient kubernetes.Interface, azureResourceClient *osmClient.Clientset, kubeController k8s.Controller) *Client {
	azureResourceFactory := osmInformers.NewSharedInformerFactory(azureResourceClient, k8s.DefaultKubeEventResyncInterval)
	informerCollection := InformerCollection{
		AzureResource: azureResourceFactory.Osm().V1().AzureResources().Informer(),
	}

	cacheCollection := CacheCollection{
		AzureResource: informerCollection.AzureResource.GetStore(),
	}

	client := Client{
		providerIdent:  kubernetesClientName,
		kubeClient:     kubeClient,
		informers:      &informerCollection,
		caches:         &cacheCollection,
		cacheSynced:    make(chan interface{}),
		announcements:  make(chan interface{}),
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
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
		azureResource := azureResourceInterface.(*osm.AzureResource)

		if !c.kubeController.IsMonitoredNamespace(azureResource.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		azureResources = append(azureResources, azureResource)
	}
	return azureResources
}

package namespace

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
)

const (
	monitorLabel = "openservicemesh.io/monitor"
)

var (
	resyncPeriod = 10 * time.Second
)

// NewNamespaceController implements namespace.Controller and creates the Kubernetes client to manage namespaces.
func NewNamespaceController(kubeConfig *rest.Config, osmID string, stop chan struct{}) Controller {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	// Only monitor namespaces that are labeled with this OSM's ID
	monitorNamespaceLabel := map[string]string{monitorLabel: osmID}
	labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, option)
	informer := informerFactory.Core().V1().Namespaces().Informer()

	client := Client{
		informer:    informer,
		cache:       informer.GetStore(),
		cacheSynced: make(chan interface{}),
	}

	informer.AddEventHandler(k8s.GetKubernetesEventHandlers("Namespaces", "Kubernetes", nil))

	if err := client.run(stop); err != nil {
		log.Fatal().Err(err).Msg("Could not start Kubernetes Namespaces client")
	}

	log.Info().Msgf("Monitoring namespaces with the label: %s=%s", monitorLabel, osmID)
	return client
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Namespace controller client started")

	if c.informer == nil {
		return errInitInformers
	}

	go c.informer.Run(stop)
	log.Info().Msgf("Waiting namespace.Monitor informer cache sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("Cache sync finished for namespace.Monitor informer")
	return nil
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c Client) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := c.cache.GetByKey(namespace)
	return exists
}

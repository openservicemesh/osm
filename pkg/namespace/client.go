package namespace

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

const (
	// MonitorLabel is the annotation label used to indicate whether a namespace should be monitored by OSM.
	MonitorLabel = "openservicemesh.io/monitored-by"
)

var (
	resyncPeriod = 30 * time.Second
)

// GetAnnouncementsChannel returns the announcement channel for the SMI client.
func (c Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

// NewNamespaceController implements namespace.Controller and creates the Kubernetes client to manage namespaces.
func NewNamespaceController(kubeClient kubernetes.Interface, meshName string, stop chan struct{}) Controller {
	// Only monitor namespaces that are labeled with this OSM's mesh name
	monitorNamespaceLabel := map[string]string{MonitorLabel: meshName}
	labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, option)
	informer := informerFactory.Core().V1().Namespaces().Informer()

	client := Client{
		informer:      informer,
		cache:         informer.GetStore(),
		cacheSynced:   make(chan interface{}),
		announcements: make(chan interface{}),
	}

	if err := client.run(stop); err != nil {
		log.Fatal().Err(err).Msg("Could not start Kubernetes Namespaces client")
	}

	informer.AddEventHandler(k8s.GetKubernetesEventHandlers("Namespace", "NamespaceClient", client.announcements, nil))

	log.Info().Msgf("Monitoring namespaces with the label: %s=%s", MonitorLabel, meshName)
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

// ListMonitoredNamespaces returns all namespaces that the mesh is monitoring.
func (c Client) ListMonitoredNamespaces() ([]string, error) {
	var namespaces []string
 
    for _, ns := range c.cache.List() {
        namespace, ok := ns.(*corev1.Namespace)
        if !ok {
			log.Error().Err(errListingNamespaces).Msg("Failed to list monitored namespaces")
            continue
        }
        namespaces = append(namespaces, namespace.Name)
    }
    return namespaces, nil
}

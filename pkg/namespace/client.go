package namespace

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	//corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/open-service-mesh/osm/pkg/log/level"
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
	var options []informers.SharedInformerOption

	// Only monitor namespaces that are labeled with this OSM's ID
	monitorNamespaceLabel := map[string]string{monitorLabel: osmID} // FIXME
	labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
	options = append(options, informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	}))
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	informerCollection := InformerCollection{
		MonitorNamespaces: informerFactory.Core().V1().Namespaces().Informer(),
	}
	cacheCollection := CacheCollection{
		MonitorNamespaces: informerCollection.MonitorNamespaces.GetStore(),
	}

	client := Client{
		informers:   &informerCollection,
		caches:      &cacheCollection,
		cacheSynced: make(chan interface{}),
	}

	h := handlers{client}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	informerCollection.MonitorNamespaces.AddEventHandler(resourceHandler)

	if err := client.run(stop); err != nil {
		glog.Fatal("Could not start Kubernetes Namespaces client", err)
	}

	return client
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	glog.V(level.Info).Infoln("Namespace controller client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"MonitorNamespaces": c.informers.MonitorNamespaces,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		glog.Info("Starting namespace informer: ", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(level.Info).Infof("Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(level.Info).Infof("Cache sync finished for %+v", names)
	return nil
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c Client) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := c.caches.MonitorNamespaces.GetByKey(namespace)
	return exists
}

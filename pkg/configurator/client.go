package configurator

import (
	"github.com/open-service-mesh/osm/pkg/client/clientset/versioned"
	"github.com/open-service-mesh/osm/pkg/client/informers/externalversions"
	"reflect"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
)

var (
	resyncPeriod = 3 * time.Second
)

// NewConfigurator implements namespace.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(kubeConfig *rest.Config, stop chan struct{}, configCRDNamespace, configCRDName string) Configurator {
	log.Info().Msgf("Watching for OSM Config CRD with name=%s in namespace=%s", configCRDName, configCRDNamespace)
	kubeClient := versioned.NewForConfigOrDie(kubeConfig)
	informerFactory := externalversions.NewSharedInformerFactory(kubeClient, resyncPeriod)
	informer := informerFactory.Osm().V1().OSMConfigs().Informer()

	client := Client{
		configCRDName:      configCRDName,
		configCRDNamespace: configCRDNamespace,
		informer:           informer,
		cache:              informer.GetStore(),
		cacheSynced:        make(chan interface{}),
		announcements:      make(chan interface{}),
	}

	if err := client.run(stop); err != nil {
		log.Fatal().Err(err).Msg("Could not start Kubernetes Configurator client")
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		name := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Name").String()
		return ns == configCRDNamespace && name == configCRDName
	}
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers("Configurator", "ConfiguratorClient", client.announcements, shouldObserve))
	return client
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Configurator controller client started")

	if c.informer == nil {
		return errInitInformers
	}

	go c.informer.Run(stop)
	log.Debug().Msgf("Waiting for OSM Configurator informer cache to sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Debug().Msgf("Cache sync finished for OSM Configurator informer")
	return nil
}

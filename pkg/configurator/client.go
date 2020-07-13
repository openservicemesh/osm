package configurator

import (
	"fmt"
	"reflect"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(kubeClient kubernetes.Interface, stop <-chan struct{}, osmNamespace, osmConfigMapName string) Configurator {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, k8s.DefaultKubeEventResyncInterval, informers.WithNamespace(osmNamespace))
	informer := informerFactory.Core().V1().ConfigMaps().Informer()
	client := Client{
		informer:         informer,
		cache:            informer.GetStore(),
		cacheSynced:      make(chan interface{}),
		announcements:    make(chan interface{}),
		osmNamespace:     osmNamespace,
		osmConfigMapName: osmConfigMapName,
	}

	// Ensure this exclusively watches only the Namespace where OSM in installed and the particular ConfigMap we need.
	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		name := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Name").String()
		return ns == osmNamespace && name == osmConfigMapName
	}

	informerName := "ConfigMap"
	providerName := "OSMConfigMap"
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers(informerName, providerName, client.announcements, shouldObserve))

	go client.run(stop)

	return &client
}

// This struct must match the shape of the "osm-config" ConfigMap
// which was created in the OSM namespace.
type osmConfig struct {
	// PermissiveTrafficPolicyMode is a bool toggle, which when TRUE ignores SMI policies and
	// allows existing Kubernetes services to communicate with each other uninterrupted.
	// This is useful whet set TRUE in brownfield configurations, where we first want to observe
	// existing traffic patterns.
	PermissiveTrafficPolicyMode bool `yaml:"permissive_traffic_policy_mode"`
}

func (c *Client) run(stop <-chan struct{}) {
	go c.informer.Run(stop)
	log.Info().Msgf("Started OSM ConfigMap informer - watching for %s", c.getConfigMapCacheKey())
	log.Info().Msg("[ConfigMap Client] Waiting for ConfigMap informer's cache to sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		log.Error().Msg("Failed initial cache sync for ConfigMap informer")
		return
	}

	// Closing the cacheSynced channel signals to the rest of the system that caches have been synced.
	close(c.cacheSynced)
	log.Info().Msg("[ConfigMap Client] Cache sync for ConfigMap informer finished")
}

func (c *Client) getConfigMapCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.osmConfigMapName)
}

func (c *Client) getConfigMap() *osmConfig {
	configMapCacheKey := c.getConfigMapCacheKey()
	item, exists, err := c.cache.GetByKey(configMapCacheKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting ConfigMap by key=%s from cache", configMapCacheKey)
	}

	if !exists {
		return &osmConfig{}
	}

	configMap := item.(*v1.ConfigMap)

	cfg := &osmConfig{}

	if modeString, ok := configMap.Data["permissive_traffic_policy_mode"]; ok {
		if modeBool, err := strconv.ParseBool(modeString); err != nil {
			log.Error().Err(err).Msgf("Error converting ConfigMap permissive_traffic_policy_mode=%+v to bool", modeString)
		} else {
			cfg.PermissiveTrafficPolicyMode = modeBool
		}
	}

	return cfg
}

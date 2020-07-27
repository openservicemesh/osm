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

const (
	permissiveTrafficPolicyModeKey = "permissive_traffic_policy_mode"
	egressKey                      = "egress"
	prometheusScrapingKey          = "prometheus_scraping"
	zipkinTracingKey               = "zipkin_tracing"
	meshCIDRRangesKey              = "mesh_cidr_ranges"
	useHTTPSIngressKey             = "use_https_ingress"
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

	client.run(stop)

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

	// Egress is a bool toggle used to enable or disable egress globally within the mesh
	Egress bool `yaml:"egress"`

	// PrometheusScraping is a bool toggle used to enable or disable metrics scraping by Prometheus
	PrometheusScraping bool `yaml:"prometheus_scraping"`

	// ZipkinTracing is a bool toggle used to enable ot disable Zipkin tracing
	ZipkinTracing bool `yaml:"zipkin_tracing"`

	// MeshCIDRRanges is the list of CIDR ranges for in-mesh traffic
	MeshCIDRRanges string `yaml:"mesh_cidr_ranges"`

	// UseHTTPSIngress is a bool toggle enabling HTTPS protocol between ingress and backend pods
	UseHTTPSIngress bool `yaml:"use_https_ingress"`
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

	var modeBool bool
	// Parse PermissiveTrafficPolicyMode
	modeBool, err = getBoolValueForKey(configMap, permissiveTrafficPolicyModeKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting value for key=%s", permissiveTrafficPolicyModeKey)
	}
	cfg.PermissiveTrafficPolicyMode = modeBool

	// Parse Egress
	modeBool, err = getBoolValueForKey(configMap, egressKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting value for key=%s", egressKey)
	}
	cfg.Egress = modeBool

	// Parse PrometheusScraping
	modeBool, err = getBoolValueForKey(configMap, prometheusScrapingKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting value for key=%s", prometheusScrapingKey)
	}
	cfg.PrometheusScraping = modeBool

	// Parse UseHTTPSIngress from ConfigMap
	modeBool, err = getBoolValueForKey(configMap, useHTTPSIngressKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting value for key=%s", useHTTPSIngressKey)
	}
	cfg.UseHTTPSIngress = modeBool

	// Parse ZipkinTracing
	modeBool, err = getBoolValueForKey(configMap, zipkinTracingKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting value for key=%s", zipkinTracingKey)
	}
	cfg.ZipkinTracing = modeBool

	// Parse MeshCIDRRanges: only required if egress is enabled
	cidr, ok := configMap.Data[meshCIDRRangesKey]
	if !ok {
		if cfg.Egress {
			log.Error().Err(errMissingKeyInConfigMap).Msgf("Missing key=%s, required when egress is enabled", meshCIDRRangesKey)
		}
	}
	cfg.MeshCIDRRanges = cidr

	return cfg
}

func getBoolValueForKey(configMap *v1.ConfigMap, key string) (bool, error) {
	modeString, ok := configMap.Data[key]
	if !ok {
		return false, errInvalidKeyInConfigMap
	}

	modeBool, err := strconv.ParseBool(modeString)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting ConfigMap key %s=%+v to bool", key, modeString)
		return false, err
	}
	return modeBool, nil
}

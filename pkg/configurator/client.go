package configurator

import (
	"fmt"
	"reflect"
	"strconv"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

const (
	permissiveTrafficPolicyModeKey = "permissive_traffic_policy_mode"
	egressKey                      = "egress"
	enableDebugServer              = "enable_debug_server"
	prometheusScrapingKey          = "prometheus_scraping"
	useHTTPSIngressKey             = "use_https_ingress"
	tracingEnableKey               = "tracing_enable"
	tracingAddressKey              = "tracing_address"
	tracingPortKey                 = "tracing_port"
	tracingEndpointKey             = "tracing_endpoint"
	envoyLogLevel                  = "envoy_log_level"
	serviceCertValidityDurationKey = "service_cert_validity_duration"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(kubeClient kubernetes.Interface, stop <-chan struct{}, osmNamespace, osmConfigMapName string) Configurator {
	return newConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
}

func newConfigurator(kubeClient kubernetes.Interface, stop <-chan struct{}, osmNamespace, osmConfigMapName string) *Client {
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

	// EnableDebugServer is a bool toggle, which enables/disables the debug server within the OSM Controller
	EnableDebugServer bool `yaml:"enable_debug_server"`

	// PrometheusScraping is a bool toggle used to enable or disable metrics scraping by Prometheus
	PrometheusScraping bool `yaml:"prometheus_scraping"`

	// UseHTTPSIngress is a bool toggle enabling HTTPS protocol between ingress and backend pods
	UseHTTPSIngress bool `yaml:"use_https_ingress"`

	// TracingEnabled is a bool toggle used to enable or disable tracing
	TracingEnable bool `yaml:"tracing_enable"`

	// TracingAddress is the address of the listener cluster
	TracingAddress string `yaml:"tracing_address"`

	// TracingPort remote port for the listener
	TracingPort int `yaml:"tracing_port"`

	// TracingEndpoint is the collector endpoint on the listener
	TracingEndpoint string `yaml:"tracing_endpoint"`

	// EnvoyLogLevel is a string that defines the log level for envoy proxies
	EnvoyLogLevel string `yaml:"envoy_log_level"`

	// ServiceCertValidityDuration is a string that defines the validity duration of service certificates
	// It is represented as a sequence of decimal numbers each with optional fraction and a unit suffix.
	// Ex: 1h to represent 1 hour, 30m to represent 30 minutes, 1.5h or 1h30m to represent 1 hour and 30 minutes.
	ServiceCertValidityDuration string `yaml:"service_cert_validity_duration"`
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
		log.Error().Err(err).Msgf("Error getting ConfigMap from cache with key %s", configMapCacheKey)
		return &osmConfig{}
	}

	if !exists {
		log.Error().Msgf("ConfigMap %s does not exist in cache", configMapCacheKey)
		return &osmConfig{}
	}

	configMap := item.(*v1.ConfigMap)

	osmConfigMap := osmConfig{
		PermissiveTrafficPolicyMode: getBoolValueForKey(configMap, permissiveTrafficPolicyModeKey),
		Egress:                      getBoolValueForKey(configMap, egressKey),
		EnableDebugServer:           getBoolValueForKey(configMap, enableDebugServer),
		PrometheusScraping:          getBoolValueForKey(configMap, prometheusScrapingKey),
		UseHTTPSIngress:             getBoolValueForKey(configMap, useHTTPSIngressKey),

		TracingEnable:               getBoolValueForKey(configMap, tracingEnableKey),
		EnvoyLogLevel:               getStringValueForKey(configMap, envoyLogLevel),
		ServiceCertValidityDuration: getStringValueForKey(configMap, serviceCertValidityDurationKey),
	}

	if osmConfigMap.TracingEnable {
		osmConfigMap.TracingAddress = getStringValueForKey(configMap, tracingAddressKey)
		osmConfigMap.TracingPort = getIntValueForKey(configMap, tracingPortKey)
		osmConfigMap.TracingEndpoint = getStringValueForKey(configMap, tracingEndpointKey)
	}

	return &osmConfigMap
}

func getBoolValueForKey(configMap *v1.ConfigMap, key string) bool {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return false
	}

	configMapBoolValue, err := strconv.ParseBool(configMapStringValue)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting ConfigMap %s/%s key %s with value %+v to bool", configMap.Namespace, configMap.Name, key, configMapStringValue)
		return false
	}

	return configMapBoolValue
}

func getIntValueForKey(configMap *v1.ConfigMap, key string) int {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return 0
	}

	configMapIntValue, err := strconv.ParseInt(configMapStringValue, 10, 32)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting ConfigMap %s/%s key %s with value %+v to integer", configMap.Namespace, configMap.Name, key, configMapStringValue)
		return 0
	}

	return int(configMapIntValue)
}

func getStringValueForKey(configMap *v1.ConfigMap, key string) string {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return ""
	}
	return configMapStringValue
}

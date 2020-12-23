package configurator

import (
	"fmt"
	"strconv"

	"github.com/cskr/pubsub"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	a "github.com/openservicemesh/osm/pkg/announcements"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

const (
	// PermissiveTrafficPolicyModeKey is the key name used for permissive mode in the ConfigMap
	PermissiveTrafficPolicyModeKey = "permissive_traffic_policy_mode"

	// egressKey is the key name used for egress in the ConfigMap
	egressKey = "egress"

	// enableDebugServer is the key name used for the debug server in the ConfigMap
	enableDebugServer = "enable_debug_server"

	// prometheusScrapingKey is the key name used for prometheus scraping in the ConfigMap
	prometheusScrapingKey = "prometheus_scraping"

	// useHTTPSIngressKey is the key name used for HTTPS ingress in the ConfigMap
	useHTTPSIngressKey = "use_https_ingress"

	// tracingEnableKey is the key name used for tracing in the ConfigMap
	tracingEnableKey = "tracing_enable"

	// tracingAddressKey is the key name used to specify the tracing address in the ConfigMap
	tracingAddressKey = "tracing_address"

	// tracingPortKey is the key name used to specify the tracing port in the ConfigMap
	tracingPortKey = "tracing_port"

	// tracingEndpointKey is the key name used to specify the tracing endpoint in the ConfigMap
	tracingEndpointKey = "tracing_endpoint"

	// envoyLogLevel is the key name used to specify the log level of Envoy proxy in the ConfigMap
	envoyLogLevel = "envoy_log_level"

	// serviceCertValidityDurationKey is the key name used to specify the validity duration of service certificates in the ConfigMap
	serviceCertValidityDurationKey = "service_cert_validity_duration"

	// defaultPubSubChannelSize is the default size of the buffered channel returned to the  subscriber
	defaultPubSubChannelSize = 128
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(kubeClient kubernetes.Interface, stop <-chan struct{}, osmNamespace, osmConfigMapName string) Configurator {
	return newConfigurator(kubeClient, stop, osmNamespace, osmConfigMapName)
}

func newConfigurator(kubeClient kubernetes.Interface, stop <-chan struct{}, osmNamespace, osmConfigMapName string) *Client {
	// Ensure this informer exclusively watches only the Namespace where OSM in installed and the particular 'osm-config' ConfigMap
	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient,
		k8s.DefaultKubeEventResyncInterval, informers.WithNamespace(osmNamespace),
		informers.WithTweakListOptions(func(listOptions *metav1.ListOptions) {
			listOptions.FieldSelector = fields.OneTermEqualSelector("metadata.name", osmConfigMapName).String()
		}))
	informer := informerFactory.Core().V1().ConfigMaps().Informer()
	client := Client{
		informer:         informer,
		cache:            informer.GetStore(),
		cacheSynced:      make(chan interface{}),
		announcements:    make(chan a.Announcement),
		osmNamespace:     osmNamespace,
		osmConfigMapName: osmConfigMapName,
		pSub:             pubsub.New(defaultPubSubChannelSize),
	}

	informerName := "ConfigMap"
	providerName := "OSMConfigMap"
	eventTypes := k8s.EventTypes{
		Add:    a.ConfigMapAdded,
		Update: a.ConfigMapUpdated,
		Delete: a.ConfigMapDeleted,
	}
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers(informerName, providerName, client.announcements, nil, nil, eventTypes))

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

// This function captures the Announcements from k8s informer updates, and relays them to the subscribed
// members on the pubsub interface
func (c *Client) publish() {
	for {
		a := <-c.announcements
		log.Debug().Msgf("OSM config publish: %s", a.Type.String())
		c.pSub.Pub(a, a.Type.String())
	}
}

func (c *Client) run(stop <-chan struct{}) {
	go c.publish()          // prepare the publish interface
	go c.informer.Run(stop) // run the informer synchronization
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

	// Parse osm-config ConfigMap.
	// In case of missing/invalid value for a key, osm-controller uses the default value.
	// Invalid values should be prevented once https://github.com/openservicemesh/osm/issues/1788
	// is implemented.
	osmConfigMap := osmConfig{}
	osmConfigMap.PermissiveTrafficPolicyMode, _ = GetBoolValueForKey(configMap, PermissiveTrafficPolicyModeKey)
	osmConfigMap.Egress, _ = GetBoolValueForKey(configMap, egressKey)
	osmConfigMap.EnableDebugServer, _ = GetBoolValueForKey(configMap, enableDebugServer)
	osmConfigMap.PrometheusScraping, _ = GetBoolValueForKey(configMap, prometheusScrapingKey)
	osmConfigMap.UseHTTPSIngress, _ = GetBoolValueForKey(configMap, useHTTPSIngressKey)
	osmConfigMap.TracingEnable, _ = GetBoolValueForKey(configMap, tracingEnableKey)
	osmConfigMap.EnvoyLogLevel, _ = GetStringValueForKey(configMap, envoyLogLevel)
	osmConfigMap.ServiceCertValidityDuration, _ = GetStringValueForKey(configMap, serviceCertValidityDurationKey)

	if osmConfigMap.TracingEnable {
		osmConfigMap.TracingAddress, _ = GetStringValueForKey(configMap, tracingAddressKey)
		osmConfigMap.TracingPort, _ = GetIntValueForKey(configMap, tracingPortKey)
		osmConfigMap.TracingEndpoint, _ = GetStringValueForKey(configMap, tracingEndpointKey)
	}

	return &osmConfigMap
}

// GetBoolValueForKey returns the boolean value for a key and an error in case of errors
func GetBoolValueForKey(configMap *v1.ConfigMap, key string) (bool, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return false, errMissingKeyInConfigMap
	}

	configMapBoolValue, err := strconv.ParseBool(configMapStringValue)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting ConfigMap %s/%s key %s with value %+v to bool", configMap.Namespace, configMap.Name, key, configMapStringValue)
		return false, err
	}

	return configMapBoolValue, nil
}

// GetIntValueForKey returns the integer value for a key and an error in case of errors
func GetIntValueForKey(configMap *v1.ConfigMap, key string) (int, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return 0, errMissingKeyInConfigMap
	}

	configMapIntValue, err := strconv.Atoi(configMapStringValue)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting ConfigMap %s/%s key %s with value %+v to integer", configMap.Namespace, configMap.Name, key, configMapStringValue)
		return 0, err
	}

	return configMapIntValue, nil
}

// GetStringValueForKey returns the string value for a key and an error in case of errors
func GetStringValueForKey(configMap *v1.ConfigMap, key string) (string, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
			key, configMap.Namespace, configMap.Name, configMap.Data)
		return "", errMissingKeyInConfigMap
	}
	return configMapStringValue, nil
}

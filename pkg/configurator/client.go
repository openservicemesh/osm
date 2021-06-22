package configurator

import (
	"fmt"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

const (
<<<<<<< HEAD
	// PermissiveTrafficPolicyModeKey is the key name used for permissive mode in the ConfigMap
	PermissiveTrafficPolicyModeKey = "permissive_traffic_policy_mode"

	// egressKey is the key name used for egress in the ConfigMap
	egressKey = "egress"

	// enableDebugServer is the key name used for the debug server in the ConfigMap
	enableDebugServer = "enable_debug_server"

	// prometheusScrapingKey is the key name used for prometheus scraping in the ConfigMap
	prometheusScrapingKey = "prometheus_scraping"

	meshCIDRRangesKey = "mesh_cidr_ranges"
	defaultInMeshCIDR = ""

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
=======
	meshConfigInformerName = "MeshConfig"
	meshConfigProviderName = "OSM"
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7

	// DefaultMeshConfigName is the default name of MeshConfig object
	DefaultMeshConfigName = "osm-mesh-config"
)

// NewConfigurator implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfigurator(kubeClient versioned.Interface, stop <-chan struct{}, osmNamespace, meshConfigName string) Configurator {
	return newConfigurator(kubeClient, stop, osmNamespace, meshConfigName)
}

func newConfigurator(meshConfigClientSet versioned.Interface, stop <-chan struct{}, osmNamespace string, meshConfigName string) *Client {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		meshConfigClientSet,
		k8s.DefaultKubeEventResyncInterval,
		informers.WithNamespace(osmNamespace),
	)
	informer := informerFactory.Config().V1alpha1().MeshConfigs().Informer()
	client := Client{
		informer:       informer,
		cache:          informer.GetStore(),
		cacheSynced:    make(chan interface{}),
		osmNamespace:   osmNamespace,
		meshConfigName: meshConfigName,
	}

	// configure listener
	eventTypes := k8s.EventTypes{
		Add:    announcements.MeshConfigAdded,
		Update: announcements.MeshConfigUpdated,
		Delete: announcements.MeshConfigDeleted,
	}
	informer.AddEventHandler(k8s.GetKubernetesEventHandlers(meshConfigInformerName, meshConfigProviderName, nil, eventTypes))

	// start listener
	go client.runMeshConfigListener(stop)

	client.run(stop)

	return &client
}

<<<<<<< HEAD
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

	// MeshCIDRRanges is the list of CIDR ranges for in-mesh traffic
	MeshCIDRRanges string `yaml:"mesh_cidr_ranges"`

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
=======
// Listens to MeshConfig events and notifies dispatcher to issue config updates to the envoys based
// on config seen on the MeshConfig
func (c *Client) runMeshConfigListener(stop <-chan struct{}) {
	// Create the subscription channel synchronously
	cfgSubChannel := events.GetPubSubInstance().Subscribe(
		announcements.MeshConfigAdded,
		announcements.MeshConfigDeleted,
		announcements.MeshConfigUpdated,
	)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7

	// Defer unsubscription on async routine exit
	defer events.GetPubSubInstance().Unsub(cfgSubChannel)

	for {
		select {
		case msg := <-cfgSubChannel:
			psubMsg, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Type assertion failed for PubSubMessage, %v\n", msg)
				continue
			}

			switch psubMsg.AnnouncementType {
			case announcements.MeshConfigAdded:
				meshConfigAddedMessageHandler(&psubMsg)
			case announcements.MeshConfigDeleted:
				meshConfigDeletedMessageHandler(&psubMsg)
			case announcements.MeshConfigUpdated:
				meshConfigUpdatedMessageHandler(&psubMsg)
			}
		case <-stop:
			log.Trace().Msgf("MeshConfig event listener exiting")
			return
		}
	}
}

func (c *Client) run(stop <-chan struct{}) {
	go c.informer.Run(stop) // run the informer synchronization
	log.Debug().Msgf("Started OSM MeshConfig informer")
	log.Debug().Msg("[MeshConfig Client] Waiting for MeshConfig informer's cache to sync")
	if !cache.WaitForCacheSync(stop, c.informer.HasSynced) {
		log.Error().Msg("Failed initial cache sync for MeshConfig informer")
		return
	}

	// Closing the cacheSynced channel signals to the rest of the system that caches have been synced.
	close(c.cacheSynced)
	log.Debug().Msg("[MeshConfig Client] Cache sync for MeshConfig informer finished")
}

func meshConfigAddedMessageHandler(psubMsg *events.PubSubMessage) {
	log.Debug().Msgf("[%s] OSM MeshConfig added event triggered a global proxy broadcast",
		psubMsg.AnnouncementType)
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.ScheduleProxyBroadcast,
		OldObj:           nil,
		NewObj:           nil,
	})
}

<<<<<<< HEAD
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
	osmConfigMap.MeshCIDRRanges = getEgressCIDR(configMap)
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

func getEgressCIDR(configMap *v1.ConfigMap) string {
	cidr, ok := configMap.Data[meshCIDRRangesKey]
	if !ok {
		return defaultInMeshCIDR
	}

	return cidr
}


// GetBoolValueForKey returns the boolean value for a key and an error in case of errors
func GetBoolValueForKey(configMap *v1.ConfigMap, key string) (bool, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		//log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
		//	key, configMap.Namespace, configMap.Name, configMap.Data)
		return false, errMissingKeyInConfigMap
=======
func meshConfigDeletedMessageHandler(psubMsg *events.PubSubMessage) {
	// Ignore deletion. We expect config to be present
	log.Debug().Msgf("[%s] OSM MeshConfig deleted event triggered a global proxy broadcast",
		psubMsg.AnnouncementType)
	events.GetPubSubInstance().Publish(events.PubSubMessage{
		AnnouncementType: announcements.ScheduleProxyBroadcast,
		OldObj:           nil,
		NewObj:           nil,
	})
}

func meshConfigUpdatedMessageHandler(psubMsg *events.PubSubMessage) {
	// Get the MeshConfig resource
	prevMeshConfig, okPrevCast := psubMsg.OldObj.(*v1alpha1.MeshConfig)
	newMeshConfig, okNewCast := psubMsg.NewObj.(*v1alpha1.MeshConfig)
	if !okPrevCast || !okNewCast {
		log.Error().Msgf("[%s] Error casting old/new MeshConfigs objects (%v %v)",
			psubMsg.AnnouncementType, okPrevCast, okNewCast)
		return
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	}

	prevSpec := prevMeshConfig.Spec
	newSpec := newMeshConfig.Spec

	// Determine if we should issue new global config update to all envoys
	triggerGlobalBroadcast := false

	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Traffic.EnableEgress != newSpec.Traffic.EnableEgress)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Traffic.EnablePermissiveTrafficPolicyMode != newSpec.Traffic.EnablePermissiveTrafficPolicyMode)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Traffic.UseHTTPSIngress != newSpec.Traffic.UseHTTPSIngress)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Observability.Tracing.Enable != newSpec.Observability.Tracing.Enable)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Observability.Tracing.Address != newSpec.Observability.Tracing.Address)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Observability.Tracing.Endpoint != newSpec.Observability.Tracing.Endpoint)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Observability.Tracing.Port != newSpec.Observability.Tracing.Port)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevSpec.Traffic.InboundExternalAuthorization.Enable != newSpec.Traffic.InboundExternalAuthorization.Enable)

	// Do not trigger updates on the inner configuration changes of ExtAuthz if disabled,
	// or otherwise skip checking if the update is to be scheduled anyway
	if newSpec.Traffic.InboundExternalAuthorization.Enable && !triggerGlobalBroadcast {
		triggerGlobalBroadcast = triggerGlobalBroadcast ||
			(prevSpec.Traffic.InboundExternalAuthorization.Address != newSpec.Traffic.InboundExternalAuthorization.Address)
		triggerGlobalBroadcast = triggerGlobalBroadcast ||
			(prevSpec.Traffic.InboundExternalAuthorization.Port != newSpec.Traffic.InboundExternalAuthorization.Port)
		triggerGlobalBroadcast = triggerGlobalBroadcast ||
			(prevSpec.Traffic.InboundExternalAuthorization.StatPrefix != newSpec.Traffic.InboundExternalAuthorization.StatPrefix)
		triggerGlobalBroadcast = triggerGlobalBroadcast ||
			(prevSpec.Traffic.InboundExternalAuthorization.Timeout != newSpec.Traffic.InboundExternalAuthorization.Timeout)
		triggerGlobalBroadcast = triggerGlobalBroadcast ||
			(prevSpec.Traffic.InboundExternalAuthorization.FailureModeAllow != newSpec.Traffic.InboundExternalAuthorization.FailureModeAllow)
	}

	if triggerGlobalBroadcast {
		log.Debug().Msgf("[%s] OSM MeshConfig update triggered global proxy broadcast",
			psubMsg.AnnouncementType)
		events.GetPubSubInstance().Publish(events.PubSubMessage{
			AnnouncementType: announcements.ScheduleProxyBroadcast,
			OldObj:           nil,
			NewObj:           nil,
		})
	} else {
		log.Trace().Msgf("[%s] OSM MeshConfig update, NOT triggering global proxy broadcast",
			psubMsg.AnnouncementType)
	}
}

<<<<<<< HEAD
// GetIntValueForKey returns the integer value for a key and an error in case of errors
func GetIntValueForKey(configMap *v1.ConfigMap, key string) (int, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		//log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
		//	key, configMap.Namespace, configMap.Name, configMap.Data)
		return 0, errMissingKeyInConfigMap
	}
=======
func (c *Client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7

// Returns the current MeshConfig
func (c *Client) getMeshConfig() *v1alpha1.MeshConfig {
	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.cache.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return &v1alpha1.MeshConfig{}
	}

<<<<<<< HEAD
	return configMapIntValue, nil
}

// GetStringValueForKey returns the string value for a key and an error in case of errors
func GetStringValueForKey(configMap *v1.ConfigMap, key string) (string, error) {
	configMapStringValue, ok := configMap.Data[key]
	if !ok {
		//log.Debug().Msgf("Key %s does not exist in ConfigMap %s/%s (%s)",
		//	key, configMap.Namespace, configMap.Name, configMap.Data)
		return "", errMissingKeyInConfigMap
=======
	var meshConfig *v1alpha1.MeshConfig
	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		meshConfig = &v1alpha1.MeshConfig{}
	} else {
		meshConfig = item.(*v1alpha1.MeshConfig)
>>>>>>> 865c66ed45ee888b5719d2e56a32f1534b61d1e7
	}

	return meshConfig
}

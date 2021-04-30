package configurator

import (
	"fmt"
	"strings"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

const (
	meshConfigInformerName = "MeshConfig"
	meshConfigProviderName = "OSM"

	// DefaultMeshConfigName is the default name of MeshConfig object
	DefaultMeshConfigName = "osm-mesh-config"
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

	// maxDataPlaneConnectionsKey is the key name used for max data plane connections in the ConfigMap
	maxDataPlaneConnectionsKey = "max_data_plane_connections"

	// tracingEnableKey is the key name used for tracing in the ConfigMap
	tracingEnableKey = "tracing_enable"

	// tracingAddressKey is the key name used to specify the tracing address in the ConfigMap
	tracingAddressKey = "tracing_address"

	// tracingPortKey is the key name used to specify the tracing port in the ConfigMap
	tracingPortKey = "tracing_port"

	// tracingEndpointKey is the key name used to specify the tracing endpoint in the ConfigMap
	tracingEndpointKey = "tracing_endpoint"

	// envoyLogLevel is the key name used to specify the log level of Envoy proxy in the ConfigMap
	envoyLogLevelKey = "envoy_log_level"

	// envoyImage is the key name used to specify the image of the Envoy proxy in the ConfigMap
	envoyImageKey = "envoy_image"

	// serviceCertValidityDurationKey is the key name used to specify the validity duration of service certificates in the ConfigMap
	serviceCertValidityDurationKey = "service_cert_validity_duration"

	// outboundIPRangeExclusionListKey is the key name used to specify the ip ranges to exclude from outbound sidecar interception
	outboundIPRangeExclusionListKey = "outbound_ip_range_exclusion_list"

	// outboundPortExclusionListKey is the key name used to specify the ports to exclude from outbound sidecar interception
	outboundPortExclusionListKey = "outbound_port_exclusion_list"

	// enablePrivilegedInitContainerKey is the key name used to specify whether init containers should be privileged in the ConfigMap
	enablePrivilegedInitContainerKey = "enable_privileged_init_container"

	// configResyncInterval is the key name used to configure the resync interval for regular proxy broadcast updates
	configResyncIntervalKey = "config_resync_interval"
)

// NewConfiguratorWithCRDClient implements configurator.Configurator and creates the Kubernetes client to manage namespaces.
func NewConfiguratorWithCRDClient(kubeClient versioned.Interface, stop <-chan struct{}, osmNamespace, meshConfigName string) Configurator {
	return newConfiguratorWithCRDClient(kubeClient, stop, osmNamespace, meshConfigName)
}

func newConfiguratorWithCRDClient(meshConfigClientSet versioned.Interface, stop <-chan struct{}, osmNamespace string, meshConfigName string) *CRDClient {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		meshConfigClientSet,
		k8s.DefaultKubeEventResyncInterval,
		informers.WithNamespace(osmNamespace),
	)
	informer := informerFactory.Config().V1alpha1().MeshConfigs().Informer()
	crdClient := CRDClient{
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
	go crdClient.runMeshConfigListener(stop)

	crdClient.run(stop)

	return &crdClient
}

// Listens to ConfigMap events and notifies dispatcher to issue config updates to the envoys based
// on config seen on the configmap
func (c *CRDClient) runMeshConfigListener(stop <-chan struct{}) {
	// Create the subscription channel synchronously
	cfgSubChannel := events.GetPubSubInstance().Subscribe(
		announcements.MeshConfigAdded,
		announcements.MeshConfigDeleted,
		announcements.MeshConfigUpdated,
	)

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

func (c *CRDClient) run(stop <-chan struct{}) {
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

// Parses a kubernetes config map object into an osm runtime object representing OSM's options/config
func parseOSMMeshConfig(meshConfig *v1alpha1.MeshConfig) *osmConfig {
	// Invalid values should be prevented once https://github.com/openservicemesh/osm/issues/1788
	// is implemented.

	spec := &meshConfig.Spec

	osmConfig := osmConfig{
		PermissiveTrafficPolicyMode:   spec.Traffic.EnablePermissiveTrafficPolicyMode,
		Egress:                        spec.Traffic.EnableEgress,
		EnableDebugServer:             spec.Observability.EnableDebugServer,
		UseHTTPSIngress:               spec.Traffic.UseHTTPSIngress,
		EnvoyLogLevel:                 spec.Sidecar.LogLevel,
		EnvoyImage:                    spec.Sidecar.EnvoyImage,
		ServiceCertValidityDuration:   spec.Certificate.ServiceCertValidityDuration,
		OutboundIPRangeExclusionList:  strings.Join(spec.Traffic.OutboundIPRangeExclusionList, ","),
		OutboundPortExclusionList:     strings.Join(spec.Traffic.OutboundPortExclusionList, ","),
		EnablePrivilegedInitContainer: spec.Sidecar.EnablePrivilegedInitContainer,
		PrometheusScraping:            spec.Observability.PrometheusScraping,
		ConfigResyncInterval:          spec.Sidecar.ConfigResyncInterval,
		MaxDataPlaneConnections:       spec.Sidecar.MaxDataPlaneConnections,
		TracingEnable:                 spec.Observability.Tracing.Enable,
	}

	if spec.Observability.Tracing.Enable {
		osmConfig.TracingAddress = spec.Observability.Tracing.Address
		osmConfig.TracingEndpoint = spec.Observability.Tracing.Endpoint
		osmConfig.TracingPort = int(spec.Observability.Tracing.Port)
	}

	return &osmConfig
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
	// Get config map
	prevMeshConfigObj, okPrevCast := psubMsg.OldObj.(*v1alpha1.MeshConfig)
	newMeshConfigObj, okNewCast := psubMsg.NewObj.(*v1alpha1.MeshConfig)
	if !okPrevCast || !okNewCast {
		log.Error().Msgf("[%s] Error casting old/new MeshConfigs objects (%v %v)",
			psubMsg.AnnouncementType, okPrevCast, okNewCast)
		return
	}

	// Parse old and new configs
	prevMeshConfig := parseOSMMeshConfig(prevMeshConfigObj)
	newMeshConfig := parseOSMMeshConfig(newMeshConfigObj)

	// Determine if we should issue new global config update to all envoys
	triggerGlobalBroadcast := false

	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.Egress != newMeshConfig.Egress)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.PermissiveTrafficPolicyMode != newMeshConfig.PermissiveTrafficPolicyMode)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.UseHTTPSIngress != newMeshConfig.UseHTTPSIngress)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.TracingEnable != newMeshConfig.TracingEnable)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.TracingAddress != newMeshConfig.TracingAddress)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.TracingEndpoint != newMeshConfig.TracingEndpoint)
	triggerGlobalBroadcast = triggerGlobalBroadcast || (prevMeshConfig.TracingPort != newMeshConfig.TracingPort)

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

func (c *CRDClient) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}

// Returns the current ConfigMap
func (c *CRDClient) getMeshConfig() *osmConfig {
	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.cache.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return &osmConfig{}
	}

	var meshConfig *v1alpha1.MeshConfig
	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		meshConfig = &v1alpha1.MeshConfig{}
	} else {
		meshConfig = item.(*v1alpha1.MeshConfig)
	}

	return parseOSMMeshConfig(meshConfig)
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

	// MaxDataPlaneConnections indicates max allowed data plane connections
	MaxDataPlaneConnections int `yaml:"max_data_plane_connections"`

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

	// EnvoyImage is the sidecar image
	EnvoyImage string `yaml:"envoy_image"`

	// ServiceCertValidityDuration is a string that defines the validity duration of service certificates
	// It is represented as a sequence of decimal numbers each with optional fraction and a unit suffix.
	// Ex: 1h to represent 1 hour, 30m to represent 30 minutes, 1.5h or 1h30m to represent 1 hour and 30 minutes.
	ServiceCertValidityDuration string `yaml:"service_cert_validity_duration"`

	// OutboundIPRangeExclusionList is the list of outbound IP ranges to exclude from sidecar interception
	OutboundIPRangeExclusionList string `yaml:"outbound_ip_range_exclusion_list"`

	// OutboundPortExclusionList is the list of outbound ports to exclude from sidecar interception
	OutboundPortExclusionList string `yaml:"outbound_port_exclusion_list"`

	EnablePrivilegedInitContainer bool `yaml:"enable_privileged_init_container"`

	// ConfigResyncInterval is a flag to configure resync interval for regular proxy broadcast updates
	ConfigResyncInterval string `yaml:"config_resync_interval"`
}

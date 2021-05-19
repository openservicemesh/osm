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
	meshConfigInformerName = "MeshConfig"
	meshConfigProviderName = "OSM"

	// DefaultMeshConfigName is the default name of MeshConfig object
	DefaultMeshConfigName = "osm-mesh-config"
)

const (
	// PermissiveTrafficPolicyModeKey is the key name used for permissive mode in the MeshConfig
	PermissiveTrafficPolicyModeKey = "permissive_traffic_policy_mode"

	// maxDataPlaneConnectionsKey is the key name used for max data plane connections in the MeshConfig
	maxDataPlaneConnectionsKey = "max_data_plane_connections"

	// tracingPortKey is the key name used to specify the tracing port in the MeshConfig
	tracingPortKey = "tracing_port"

	// envoyLogLevel is the key name used to specify the log level of Envoy proxy in the MeshConfig
	envoyLogLevelKey = "envoy_log_level"

	// envoyImage is the key name used to specify the image of the Envoy proxy in the MeshConfig
	envoyImageKey = "envoy_image"

	// serviceCertValidityDurationKey is the key name used to specify the validity duration of service certificates in the MeshConfig
	serviceCertValidityDurationKey = "service_cert_validity_duration"

	// outboundIPRangeExclusionListKey is the key name used to specify the ip ranges to exclude from outbound sidecar interception
	outboundIPRangeExclusionListKey = "outbound_ip_range_exclusion_list"

	// outboundPortExclusionListKey is the key name used to specify the ports to exclude from outbound sidecar interception
	outboundPortExclusionListKey = "outbound_port_exclusion_list"

	// configResyncInterval is the key name used to configure the resync interval for regular proxy broadcast updates
	configResyncIntervalKey = "config_resync_interval"
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

// Listens to MeshConfig events and notifies dispatcher to issue config updates to the envoys based
// on config seen on the MeshConfig
func (c *Client) runMeshConfigListener(stop <-chan struct{}) {
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

func (c *Client) getMeshConfigCacheKey() string {
	return fmt.Sprintf("%s/%s", c.osmNamespace, c.meshConfigName)
}

// Returns the current MeshConfig
func (c *Client) getMeshConfig() *v1alpha1.MeshConfig {
	meshConfigCacheKey := c.getMeshConfigCacheKey()
	item, exists, err := c.cache.GetByKey(meshConfigCacheKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting MeshConfig from cache with key %s", meshConfigCacheKey)
		return &v1alpha1.MeshConfig{}
	}

	var meshConfig *v1alpha1.MeshConfig
	if !exists {
		log.Warn().Msgf("MeshConfig %s does not exist. Default config values will be used.", meshConfigCacheKey)
		meshConfig = &v1alpha1.MeshConfig{}
	} else {
		meshConfig = item.(*v1alpha1.MeshConfig)
	}

	return meshConfig
}

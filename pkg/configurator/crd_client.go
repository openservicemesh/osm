package configurator

import (
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
)

func newConfiguratorWithCRDClient(meshConfigClientSet versioned.Interface, stop <-chan struct{}, osmNamespace string) *CRDClient {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		meshConfigClientSet,
		k8s.DefaultKubeEventResyncInterval,
		informers.WithNamespace(osmNamespace),
	)
	informer := informerFactory.Config().V1alpha1().MeshConfigs().Informer()
	crdClient := CRDClient{
		informer:     informer,
		cache:        informer.GetStore(),
		cacheSynced:  make(chan interface{}),
		osmNamespace: osmNamespace,
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

	// start informer
	go informer.Run(stop)

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

// TODO: remove nolint after invocation implemented
func (c *CRDClient) run(stop <-chan struct{}) { //nolint:golint,unused
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
	//
	// Unsupported fields in MeshConfig CRD:
	// * PrometheusScraping
	// * ConfigResyncInterval

	osmConfig := osmConfig{}
	osmConfig.PermissiveTrafficPolicyMode = meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode
	osmConfig.Egress = meshConfig.Spec.Traffic.EnableEgress
	osmConfig.EnableDebugServer = meshConfig.Spec.Observability.EnableDebugServer
	osmConfig.UseHTTPSIngress = meshConfig.Spec.Traffic.UseHTTPSIngress
	osmConfig.TracingEnable = meshConfig.Spec.Observability.Tracing.Enable
	osmConfig.EnvoyLogLevel = meshConfig.Spec.Sidecar.LogLevel
	osmConfig.EnvoyImage = meshConfig.Spec.Sidecar.EnvoyImage
	osmConfig.InitContainerImage = meshConfig.Spec.Sidecar.InitContainerImage
	osmConfig.ServiceCertValidityDuration = meshConfig.Spec.Certificate.ServiceCertValidityDuration
	osmConfig.OutboundIPRangeExclusionList = strings.Join(meshConfig.Spec.Traffic.OutboundIPRangeExclusionList, ",")
	osmConfig.OutboundPortExclusionList = strings.Join(meshConfig.Spec.Traffic.OutboundPortExclusionList, ",")
	osmConfig.EnablePrivilegedInitContainer = meshConfig.Spec.Sidecar.EnablePrivilegedInitContainer

	if osmConfig.TracingEnable {
		osmConfig.TracingAddress = meshConfig.Spec.Observability.Tracing.Address
		osmConfig.TracingPort = int(meshConfig.Spec.Observability.Tracing.Port)
		osmConfig.TracingEndpoint = meshConfig.Spec.Observability.Tracing.Endpoint
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

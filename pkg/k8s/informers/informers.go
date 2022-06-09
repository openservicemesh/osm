package informers

import (
	"github.com/rs/zerolog/log"
	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
)

type InformerCollectionOpt func(*InformerCollection)

func NewInformerCollection(meshName string, stop <-chan struct{}, kubeClient kubernetes.Interface, smiTrafficSplitClient smiTrafficSplitClient.Interface,
	smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiTrafficAccessClient.Interface, configClient configClientset.Interface,
	policyClient policyClientset.Interface, opts ...InformerCollectionOpt) (*InformerCollection, error) {

	ic := &InformerCollection{
		meshName:              meshName,
		kubeClient:            kubeClient,
		smiTrafficSplitClient: smiTrafficSplitClient,
		smiTrafficSpecClient:  smiTrafficSpecClient,
		smiAccessClient:       smiAccessClient,
		configClient:          configClient,
		policyClient:          policyClient,
		informers:             make(informerCollection),
	}

	for _, opt := range opts {
		opt(ic)
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]informerInit{
		// Kubernetes
		InformerKeyNamespace:      ic.initNamespaceMonitor,
		InformerKeyService:        ic.initServicesMonitor,
		InformerKeyServiceAccount: ic.initServiceAccountsMonitor,
		InformerKeyPod:            ic.initPodMonitor,
		InformerKeyEndpoints:      ic.initEndpointMonitor,

		// SMI
		InformerKeyTrafficSplit:   ic.initTrafficSplitMonitor,
		InformerKeyTrafficTarget:  ic.initTrafficTargetMonitor,
		InformerKeyHTTPRouteGroup: ic.initHTTPRouteGroupMonitor,
		InformerKeyTCPRoute:       ic.initTCPRouteMonitor,

		// Config
		InformerKeyMeshConfig:          ic.initMeshConfigMonitor,
		InformerKeyMeshRootCertificate: ic.initMeshRootCertificateMonitor,

		// Policy
		InformerKeyEgress:                 ic.initEgressMonitor,
		InformerKeyIngressBackend:         ic.initIngressBackendMonitor,
		InformerKeyUpstreamTrafficSetting: ic.initUpstreamTrafficSettingMonitor,
		InformerKeyRetry:                  ic.initRetryMonitor,
	}

	if len(ic.selectedInformers) == 0 {
		// Initialize all informers
		ic.selectedInformers = []InformerKey{
			InformerKeyNamespace,
			InformerKeyService,
			InformerKeyPod,
			InformerKeyEndpoints,
			InformerKeyServiceAccount,
			InformerKeyTrafficSplit,
			InformerKeyTrafficTarget,
			InformerKeyHTTPRouteGroup,
			InformerKeyTCPRoute,
			InformerKeyMeshConfig,
			InformerKeyMeshRootCertificate,
			InformerKeyEgress,
			InformerKeyIngressBackend,
			InformerKeyUpstreamTrafficSetting,
			InformerKeyRetry,
		}
	}

	for _, key := range ic.selectedInformers {
		informerInitHandlerMap[key]()
	}

	if err := ic.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return ic, nil
}

// WithCustomStore sets the shared store for the informerCollection to reference.
// This store will be used instead of each informer's store. This should typically
// only be used in tests
func WithCustomStores(stores map[InformerKey]cache.Store) InformerCollectionOpt {
	return func(ic *InformerCollection) {
		ic.customStores = stores
	}
}

func WithSelectedInformers(selectedInformers ...InformerKey) InformerCollectionOpt {
	return func(ic *InformerCollection) {
		ic.selectedInformers = selectedInformers
	}
}

func (ic *InformerCollection) run(stop <-chan struct{}) error {
	log.Info().Msg("Namespace controller client started")
	var hasSynced []cache.InformerSynced
	var names []string

	if ic.informers == nil {
		return errInitInformers
	}

	for name, informer := range ic.informers {
		if informer == nil {
			continue
		}

		go informer.Run(make(chan struct{}))
		names = append(names, string(name))
		log.Info().Msgf("Waiting for %s informer cache sync...", name)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Caches for %v synced successfully", names)

	return nil
}

func (ic *InformerCollection) AddEventHandler(informerKey InformerKey, handler cache.ResourceEventHandler) {
	i, ok := ic.informers[informerKey]
	if !ok {
		log.Info().Msgf("attempted to add event handler for nil informer %s", informerKey)
		return
	}

	i.informer.AddEventHandler(handler)
}

func (ic *InformerCollection) GetByKey(informerKey InformerKey, objectKey string) (interface{}, bool, error) {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil, false, nil
	}

	return informer.GetStore().GetByKey(objectKey)
}

func (ic *InformerCollection) List(informerKey InformerKey) []interface{} {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil
	}

	return informer.GetStore().List()
}

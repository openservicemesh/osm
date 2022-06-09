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

var (
	k8sInformerKeys = []InformerKey{
		InformerKeyNamespace,
		InformerKeyService,
		InformerKeyServiceAccount,
		InformerKeyPod,
		InformerKeyEndpoints,
	}

	smiInformerKeys = []InformerKey{
		InformerKeyTrafficSplit,
		InformerKeyTrafficTarget,
		InformerKeyHTTPRouteGroup,
		InformerKeyTCPRoute,
	}

	configInformerKeys = []InformerKey{
		InformerKeyMeshConfig,
		InformerKeyMeshRootCertificate,
	}

	policyInformerKeys = []InformerKey{
		InformerKeyEgress,
		InformerKeyIngressBackend,
		InformerKeyUpstreamTrafficSetting,
		InformerKeyRetry,
	}
)

// InformerCollectionOption is a function that modifies an informer collection
type InformerCollectionOption func(*InformerCollection)

// NewInformerCollection creates a new InformerCollection
func NewInformerCollection(meshName string, stop <-chan struct{}, opts ...InformerCollectionOption) (*InformerCollection, error) {
	ic := &InformerCollection{
		meshName:          meshName,
		informers:         map[InformerKey]*informer{},
		selectedInformers: map[InformerKey]struct{}{},
	}

	// Execute all of the given options (e.g. set clients, set custom stores, etc.)
	for _, opt := range opts {
		if opt != nil {
			opt(ic)
		}
	}

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
		// Select all informers
		ic.selectedInformers = map[InformerKey]struct{}{
			InformerKeyNamespace:              {},
			InformerKeyService:                {},
			InformerKeyPod:                    {},
			InformerKeyEndpoints:              {},
			InformerKeyServiceAccount:         {},
			InformerKeyTrafficSplit:           {},
			InformerKeyTrafficTarget:          {},
			InformerKeyHTTPRouteGroup:         {},
			InformerKeyTCPRoute:               {},
			InformerKeyMeshConfig:             {},
			InformerKeyMeshRootCertificate:    {},
			InformerKeyEgress:                 {},
			InformerKeyIngressBackend:         {},
			InformerKeyUpstreamTrafficSetting: {},
			InformerKeyRetry:                  {},
		}
	}

	for key := range ic.selectedInformers {
		informerInitHandlerMap[key]()
	}

	if err := ic.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return ic, nil
}

// WithCustomStores provides the InformerCollection an set of `cache.Store`s indexed
// by InformerKey. This functionality was added for the express purpose of testing
// flexibility since the alternative often leads to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`.
func WithCustomStores(stores map[InformerKey]cache.Store) InformerCollectionOption {
	return func(ic *InformerCollection) {
		ic.customStores = stores
	}
}

// WithKubeClient sets the kubeClient for the InformerCollection
func WithKubeClient(kubeClient kubernetes.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		ic.kubeClient = kubeClient

		// select the k8s informers
		for _, key := range k8sInformerKeys {
			ic.selectedInformers[key] = struct{}{}
		}
	}
}

// WithSMIClients sets the SMI clients for the InformerCollection
func WithSMIClients(smiTrafficSplitClient smiTrafficSplitClient.Interface, smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiTrafficAccessClient.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		ic.smiTrafficSplitClient = smiTrafficSplitClient
		ic.smiTrafficSpecClient = smiTrafficSpecClient
		ic.smiAccessClient = smiAccessClient

		// select the SMI informers
		for _, key := range smiInformerKeys {
			ic.selectedInformers[key] = struct{}{}
		}
	}
}

// WithConfigClient sets the config client for the InformerCollection
func WithConfigClient(configClient configClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		ic.configClient = configClient

		// select the config informers
		for _, key := range configInformerKeys {
			ic.selectedInformers[key] = struct{}{}
		}
	}
}

// WithPolicyClient sets the policy client for the InformerCollection
func WithPolicyClient(policyClient policyClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		ic.policyClient = policyClient

		// select the policy informers
		for _, key := range policyInformerKeys {
			ic.selectedInformers[key] = struct{}{}
		}
	}
}

func (ic *InformerCollection) run(stop <-chan struct{}) error {
	log.Info().Msg("InformerCollection started")
	var hasSynced []cache.InformerSynced
	var names []string

	if ic.informers == nil {
		return errInitInformers
	}

	for name, informer := range ic.informers {
		if informer == nil {
			continue
		}

		go informer.Run(stop)
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

// AddEventHandler adds an handler to the informer indexed by the given InformerKey
func (ic *InformerCollection) AddEventHandler(informerKey InformerKey, handler cache.ResourceEventHandler) {
	i, ok := ic.informers[informerKey]
	if !ok {
		log.Info().Msgf("attempted to add event handler for nil informer %s", informerKey)
		return
	}

	i.informer.AddEventHandler(handler)
}

// GetByKey retrieves an item (based on the given index) from the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) GetByKey(informerKey InformerKey, objectKey string) (interface{}, bool, error) {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil, false, nil
	}

	return informer.GetStore().GetByKey(objectKey)
}

// List returns the contents of the store of the informer indexed by the given InformerKey
func (ic *InformerCollection) List(informerKey InformerKey) []interface{} {
	informer, ok := ic.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil
	}

	return informer.GetStore().List()
}

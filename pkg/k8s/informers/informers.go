package informers

import (
	"errors"
	"reflect"
	"testing"

	"github.com/rs/zerolog/log"
	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiAccessInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configInformers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	policyInformers "github.com/openservicemesh/osm/pkg/gen/client/policy/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

// InformerCollectionOption is a function that modifies an informer collection
type InformerCollectionOption func(*InformerCollection)

// NewInformerCollection creates a new InformerCollection
func NewInformerCollection(meshName string, broker *messaging.Broker, stop <-chan struct{}, opts ...InformerCollectionOption) (*InformerCollection, error) {
	ic := &InformerCollection{
		meshName:  meshName,
		broker:    broker,
		informers: map[InformerKey]cache.SharedIndexInformer{},
	}

	// Execute all of the given options (e.g. set clients, set custom stores, etc.)
	for _, opt := range opts {
		if opt != nil {
			opt(ic)
		}
	}

	for _, informer := range ic.informers {
		// add eventhandler
		informer.AddEventHandler(ic.eventHandlers())
	}

	if err := ic.run(stop); err != nil {
		log.Error().Err(err).Msg("Could not start informer collection")
		return nil, err
	}

	return ic, nil
}

// WithKubeClient sets the kubeClient for the InformerCollection
func WithKubeClient(kubeClient kubernetes.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		// initialize informers
		monitorNamespaceLabel := map[string]string{constants.OSMKubeResourceMonitorAnnotation: ic.meshName}

		labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
		option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.LabelSelector = labelSelector
		})

		nsInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, DefaultKubeEventResyncInterval, option)
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)
		v1api := informerFactory.Core().V1()
		ic.informers[InformerKeyNamespace] = nsInformerFactory.Core().V1().Namespaces().Informer()
		ic.informers[InformerKeyService] = v1api.Services().Informer()
		ic.informers[InformerKeyServiceAccount] = v1api.ServiceAccounts().Informer()
		ic.informers[InformerKeyPod] = v1api.Pods().Informer()
		ic.informers[InformerKeyEndpoints] = v1api.Endpoints().Informer()
	}
}

// WithSMIClients sets the SMI clients for the InformerCollection
func WithSMIClients(smiTrafficSplitClient smiTrafficSplitClient.Interface, smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiTrafficAccessClient.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		accessInformerFactory := smiAccessInformers.NewSharedInformerFactory(smiAccessClient, DefaultKubeEventResyncInterval)
		splitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, DefaultKubeEventResyncInterval)
		specInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyTCPRoute] = specInformerFactory.Specs().V1alpha4().TCPRoutes().Informer()
		ic.informers[InformerKeyHTTPRouteGroup] = specInformerFactory.Specs().V1alpha4().HTTPRouteGroups().Informer()
		ic.informers[InformerKeyTrafficTarget] = accessInformerFactory.Access().V1alpha3().TrafficTargets().Informer()
		ic.informers[InformerKeyTrafficSplit] = splitInformerFactory.Split().V1alpha2().TrafficSplits().Informer()
	}
}

// WithConfigClient sets the config client for the InformerCollection
func WithConfigClient(configClient configClientset.Interface, meshConfigName, osmNamespace string) InformerCollectionOption {
	return func(ic *InformerCollection) {
		listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, meshConfigName).String()
		})
		meshConfiginformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(osmNamespace), listOption)
		mrcInformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(osmNamespace))

		ic.informers[InformerKeyMeshConfig] = meshConfiginformerFactory.Config().V1alpha2().MeshConfigs().Informer()
		ic.informers[InformerKeyMeshRootCertificate] = mrcInformerFactory.Config().V1alpha2().MeshRootCertificates().Informer()
	}
}

// WithPolicyClient sets the policy client for the InformerCollection
func WithPolicyClient(policyClient policyClientset.Interface) InformerCollectionOption {
	return func(ic *InformerCollection) {
		informerFactory := policyInformers.NewSharedInformerFactory(policyClient, DefaultKubeEventResyncInterval)

		ic.informers[InformerKeyEgress] = informerFactory.Policy().V1alpha1().Egresses().Informer()
		ic.informers[InformerKeyIngressBackend] = informerFactory.Policy().V1alpha1().IngressBackends().Informer()
		ic.informers[InformerKeyUpstreamTrafficSetting] = informerFactory.Policy().V1alpha1().UpstreamTrafficSettings().Informer()
		ic.informers[InformerKeyRetry] = informerFactory.Policy().V1alpha1().Retries().Informer()
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

// Add is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Add(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to add to nil store with key %s", key)
	}

	return i.GetStore().Add(obj)
}

// Update is only exported for the sake of tests and requires a testing.T to ensure it's
// never used in production. This functionality was added for the express purpose of testing
// flexibility since the alternatives can often lead to flaky tests and race conditions
// between the time an object is added to a fake clientset and when that object
// is actually added to the informer `cache.Store`
func (ic *InformerCollection) Update(key InformerKey, obj interface{}, t *testing.T) error {
	if t == nil {
		return errors.New("this method should only be used in tests")
	}

	i, ok := ic.informers[key]
	if !ok {
		t.Errorf("tried to update to nil store with key %s", key)
	}

	return i.GetStore().Update(obj)
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

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (ic *InformerCollection) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := ic.informers[InformerKeyNamespace].GetStore().GetByKey(namespace)
	return exists
}

// AddEventHandler adds an handler to the informer indexed by the given InformerKey
func (ic *InformerCollection) AddEventHandler(informerKey InformerKey, handler cache.ResourceEventHandler) {
	i, ok := ic.informers[informerKey]
	if !ok {
		log.Info().Msgf("attempted to add event handler for nil informer %s", informerKey)
		return
	}

	i.AddEventHandler(handler)
}

func (ic *InformerCollection) eventHandlers() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ic.queueEvent(obj, events.PubSubMessage{
				Type:   events.Added,
				Kind:   events.GetKind(obj),
				NewObj: obj,
			})
		},
		UpdateFunc: func(old, new interface{}) {
			ic.queueEvent(new, events.PubSubMessage{
				Type:   events.Updated,
				Kind:   events.GetKind(new),
				OldObj: old,
				NewObj: new,
			})
		},
		DeleteFunc: func(obj interface{}) {
			ic.queueEvent(obj, events.PubSubMessage{
				Type:   events.Deleted,
				Kind:   events.GetKind(obj),
				OldObj: obj,
			})
		},
	}
}

func (ic *InformerCollection) shouldObserve(obj interface{}) bool {
	switch obj.(type) {
	case *corev1.Namespace, *configv1alpha2.MeshConfig, *configv1alpha2.MeshRootCertificate:
		return true
	default:
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return ic.IsMonitoredNamespace(object.GetNamespace())
	}
}

func (ic *InformerCollection) queueEvent(obj interface{}, event events.PubSubMessage) {
	if !ic.shouldObserve(obj) {
		return
	}
	logResourceEvent(event.Topic(), obj)
	ns := getNamespace(obj)
	metricsstore.DefaultMetricsStore.K8sAPIEventCounter.WithLabelValues(event.Topic(), ns).Inc()
	ic.broker.GetQueue().AddRateLimited(event)
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logResourceEvent(event string, obj interface{}) {
	log := log.With().Str("event", event).Logger()
	o, err := meta.Accessor(obj)
	if err != nil {
		log.Error().Err(err).Msg("error parsing object, ignoring")
		return
	}
	name := o.GetName()
	if o.GetNamespace() != "" {
		name = o.GetNamespace() + "/" + name
	}
	log.Debug().Str("resource_name", name).Msg("received kubernetes resource event")
}

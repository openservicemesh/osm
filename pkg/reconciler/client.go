package reconciler

import (
	"strconv"

	clientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	customResourceDefinitionInformer "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/apiextensions/v1"
	internalinterfaces "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions/internalinterfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
)

// NewReconcilerClient implements a client to reconcile osm managed resources
func NewReconcilerClient(kubeClient kubernetes.Interface, apiServerClient clientset.Interface, meshName, osmVersion string, stop chan struct{}, selectInformers ...k8s.InformerKey) error {
	// Initialize client object
	c := client{
		kubeClient:      kubeClient,
		meshName:        meshName,
		osmVersion:      osmVersion,
		apiServerClient: apiServerClient,
		informers:       informerCollection{},
	}

	// Initialize informers
	informerInitHandlerMap := map[k8s.InformerKey]func(){
		CrdInformerKey:               c.initCustomResourceDefinitionMonitor,
		MutatingWebhookInformerKey:   c.initMutatingWebhookConfigurationMonitor,
		ValidatingWebhookInformerKey: c.initValidatingWebhookConfigurationMonitor,
	}

	// If specific informers are not selected to be initialized, initialize all informers
	if len(selectInformers) == 0 {
		informers := []k8s.InformerKey{MutatingWebhookInformerKey, ValidatingWebhookInformerKey}
		// initialize informer for CRDs only if the apiServerClient is not nil
		if apiServerClient != nil {
			informers = append(informers, CrdInformerKey)
		}
		selectInformers = informers
	}

	for _, informer := range selectInformers {
		informerInitHandlerMap[informer]()
	}

	if err := c.run(stop); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrStartingReconciler)).
			Msg("Could not start osm reconciler client")
		return err
	}

	return nil
}

// Initializes CustomResourceDefinition monitoring
func (c *client) initCustomResourceDefinitionMonitor() {
	osmCrdsLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.ReconcileLabel: strconv.FormatBool(true)}

	labelSelector := fields.SelectorFromSet(osmCrdsLabel).String()
	options := internalinterfaces.TweakListOptionsFunc(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := customResourceDefinitionInformer.NewFilteredCustomResourceDefinitionInformer(c.apiServerClient, k8s.DefaultKubeEventResyncInterval, cache.Indexers{nameIndex: metaNameIndexFunc}, options)

	// Add informer
	c.informers[CrdInformerKey] = informerFactory

	// Add event handler to informer
	c.informers[CrdInformerKey].AddEventHandler(c.crdEventHandler())
}

// Initializes mutating webhook monitoring
func (c *client) initMutatingWebhookConfigurationMonitor() {
	osmMwhcLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.OSMAppInstanceLabelKey: c.meshName, constants.ReconcileLabel: strconv.FormatBool(true)}
	labelSelector := fields.SelectorFromSet(osmMwhcLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, k8s.DefaultKubeEventResyncInterval, option)

	// Add informer
	c.informers[MutatingWebhookInformerKey] = informerFactory.Admissionregistration().V1().MutatingWebhookConfigurations().Informer()

	// Add event handler to informer
	c.informers[MutatingWebhookInformerKey].AddEventHandler(c.mutatingWebhookEventHandler())
}

// Initializes validating webhook monitoring
func (c *client) initValidatingWebhookConfigurationMonitor() {
	osmVwhcLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.OSMAppInstanceLabelKey: c.meshName, constants.ReconcileLabel: strconv.FormatBool(true)}
	labelSelector := fields.SelectorFromSet(osmVwhcLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, k8s.DefaultKubeEventResyncInterval, option)

	// Add informer
	c.informers[ValidatingWebhookInformerKey] = informerFactory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer()

	// Add event handler to informer
	c.informers[ValidatingWebhookInformerKey].AddEventHandler(c.validatingWebhookEventHandler())
}

func (c *client) run(stop <-chan struct{}) error {
	log.Info().Msg("OSM reconciler client started")
	var hasSynced []cache.InformerSynced
	var names []string

	if c.informers == nil {
		log.Error().Err(errInitInformers).Msg("No resources added to reconciler's informer")
		return errInitInformers
	}

	for name, informer := range c.informers {
		if informer == nil {
			continue
		}

		log.Info().Msgf("Starting reconciler informer: %s", name)
		go informer.Run(stop)
		names = append(names, (string)(name))
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("Waiting for reconciler informer's cache to sync")
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for reconciler informer : %v", names)
	return nil
}

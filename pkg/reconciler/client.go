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
func NewReconcilerClient(kubeClient kubernetes.Interface, apiServerClient clientset.Interface, meshName, osmVersion string, stop chan struct{}, selectInformers ...InformerKey) error {
	// Initialize client object
	c := client{
		kubeClient:      kubeClient,
		meshName:        meshName,
		osmVersion:      osmVersion,
		apiServerClient: apiServerClient,
		informers:       informerCollection{},
	}

	// Initialize informers
	informerInitHandlerMap := map[InformerKey]func() error{
		CrdInformerKey:               c.initCustomResourceDefinitionMonitor,
		MutatingWebhookInformerKey:   c.initMutatingWebhookConfigurationMonitor,
		ValidatingWebhookInformerKey: c.initValidatingWebhookConfigurationMonitor,
	}

	for _, informer := range selectInformers {
		if err := informerInitHandlerMap[informer](); err != nil {
			log.Error().Err(err).Msgf("Failed to initialize informer for %s", informer)
			return err
		}
	}

	if err := c.run(stop); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrStartingReconciler)).
			Msg("Could not start osm reconciler client")
		return err
	}

	return nil
}

// Initializes CustomResourceDefinition monitoring
// Returns an error if the informer could not be initialized
func (c *client) initCustomResourceDefinitionMonitor() error {
	// Use the OSM version as the selector for reconciliation
	osmCrdsLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.ReconcileLabel: c.osmVersion}

	labelSelector := fields.SelectorFromSet(osmCrdsLabel).String()
	options := internalinterfaces.TweakListOptionsFunc(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := customResourceDefinitionInformer.NewFilteredCustomResourceDefinitionInformer(c.apiServerClient, k8s.DefaultKubeEventResyncInterval, cache.Indexers{nameIndex: metaNameIndexFunc}, options)

	// Add informer
	c.informers[CrdInformerKey] = informerFactory

	// Add event handler to informer
	if _, err := c.informers[CrdInformerKey].AddEventHandler(c.crdEventHandler()); err != nil {
		log.Error().Err(err).Msgf("Failed to add event handler to informer %s", CrdInformerKey)
		return err
	}
	return nil
}

// Initializes mutating webhook monitoring
// Returns an error if the informer could not be initialized
func (c *client) initMutatingWebhookConfigurationMonitor() error {
	osmMwhcLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.OSMAppInstanceLabelKey: c.meshName, constants.ReconcileLabel: strconv.FormatBool(true)}
	labelSelector := fields.SelectorFromSet(osmMwhcLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, k8s.DefaultKubeEventResyncInterval, option)

	// Add informer
	c.informers[MutatingWebhookInformerKey] = informerFactory.Admissionregistration().V1().MutatingWebhookConfigurations().Informer()

	// Add event handler to informer
	if _, err := c.informers[MutatingWebhookInformerKey].AddEventHandler(c.mutatingWebhookEventHandler()); err != nil {
		log.Error().Err(err).Msgf("Failed to add event handler to informer %s", MutatingWebhookInformerKey)
		return err
	}
	return nil
}

// Initializes validating webhook monitoring
// Returns an error if the informer could not be initialized
func (c *client) initValidatingWebhookConfigurationMonitor() error {
	osmVwhcLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue, constants.OSMAppInstanceLabelKey: c.meshName, constants.ReconcileLabel: strconv.FormatBool(true)}
	labelSelector := fields.SelectorFromSet(osmVwhcLabel).String()
	option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
		opt.LabelSelector = labelSelector
	})

	informerFactory := informers.NewSharedInformerFactoryWithOptions(c.kubeClient, k8s.DefaultKubeEventResyncInterval, option)

	// Add informer
	c.informers[ValidatingWebhookInformerKey] = informerFactory.Admissionregistration().V1().ValidatingWebhookConfigurations().Informer()

	// Add event handler to informer
	if _, err := c.informers[ValidatingWebhookInformerKey].AddEventHandler(c.validatingWebhookEventHandler()); err != nil {
		log.Error().Err(err).Msgf("Failed to add event handler to informer %s", ValidatingWebhookInformerKey)
		return err
	}
	return nil
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

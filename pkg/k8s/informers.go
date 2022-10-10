package k8s

import (
	"time"

	"errors"

	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiAccessInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	mcsClient "sigs.k8s.io/mcs-api/pkg/client/clientset/versioned"
	mcsInformers "sigs.k8s.io/mcs-api/pkg/client/informers/externalversions"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configInformers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions"
	policyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned"
	policyInformers "github.com/openservicemesh/osm/pkg/gen/client/policy/informers/externalversions"
)

// informerKey stores the different Informers we keep for K8s resources
type informerKey string

const (
	// informerKeyNamespace is the informerKey for a Namespace informer
	informerKeyNamespace informerKey = "Namespace"
	// informerKeyService is the informerKey for a Service informer
	informerKeyService informerKey = "Service"
	// informerKeyPod is the informerKey for a Pod informer
	informerKeyPod informerKey = "Pod"
	// informerKeyEndpoints is the informerKey for a Endpoints informer
	informerKeyEndpoints informerKey = "Endpoints"
	// informerKeyServiceAccount is the informerKey for a ServiceAccount informer
	informerKeyServiceAccount informerKey = "ServiceAccount"
	// informerKeySecret is the informerKey for a Secret informer
	informerKeySecret informerKey = "Secret"

	// informerKeyTrafficSplit is the informerKey for a TrafficSplit informer
	informerKeyTrafficSplit informerKey = "TrafficSplit"
	// informerKeyTrafficTarget is the informerKey for a TrafficTarget informer
	informerKeyTrafficTarget informerKey = "TrafficTarget"
	// informerKeyHTTPRouteGroup is the informerKey for a HTTPRouteGroup informer
	informerKeyHTTPRouteGroup informerKey = "HTTPRouteGroup"
	// informerKeyTCPRoute is the informerKey for a TCPRoute informer
	informerKeyTCPRoute informerKey = "TCPRoute"

	// informerKeyMeshConfig is the informerKey for a MeshConfig informer
	informerKeyMeshConfig informerKey = "MeshConfig"
	// informerKeyMeshRootCertificate is the informerKey for a MeshRootCertificate informer
	informerKeyMeshRootCertificate informerKey = "MeshRootCertificate"

	// informerKeyEgress is the informerKey for a Egress informer
	informerKeyEgress informerKey = "Egress"
	// informerKeyIngressBackend is the informerKey for a IngressBackend informer
	informerKeyIngressBackend informerKey = "IngressBackend"
	// informerKeyUpstreamTrafficSetting is the informerKey for a UpstreamTrafficSetting informer
	informerKeyUpstreamTrafficSetting informerKey = "UpstreamTrafficSetting"
	// informerKeyRetry is the informerKey for a Retry informer
	informerKeyRetry informerKey = "Retry"
	// informerKeyTelemetry lookup identifier
	informerKeyTelemetry informerKey = "Telemetry"
	// informerKeyExtensionService is the informerKey for an ExtensionService informer
	informerKeyExtensionService informerKey = "ExtensionService"
	// informerKeyServiceImport is the informerKey for a ServiceImport informer
	informerKeyServiceImport informerKey = "ServiceImport"
	// informerKeyServiceExport is the informerKey for a ServiceExport informer
	informerKeyServiceExport informerKey = "ServiceExport"
)

const (
	// DefaultKubeEventResyncInterval is the default resync interval for k8s events
	// This is set to 0 because we do not need resyncs from k8s client.
	// For the MeshConfig resource, we have our own Ticker to turn on periodic resyncs.
	DefaultKubeEventResyncInterval = 0 * time.Second
	// MRCResyncInterval is the resync interval to provide a retry mechanism for MRC event handlers
	// It is half of the MrcDurationPerStage (the amount of time we leave each MRC in a stage before moving to the next stage)
	MRCResyncInterval = certificate.MrcDurationPerStage / 2
)

var (
	errInitInformers = errors.New("informer not initialized")
	errSyncingCaches = errors.New("failed initial cache sync for informers")
)

// ClientOption is a function that modifies an informer collection
type ClientOption func(*Client)

// WithKubeClient sets the kubeClient for the Client
func WithKubeClient(kubeClient kubernetes.Interface, meshName string) ClientOption {
	return func(c *Client) {
		c.kubeClient = kubeClient
		// initialize informers
		monitorNamespaceLabel := map[string]string{constants.OSMKubeResourceMonitorAnnotation: meshName}

		labelSelector := fields.SelectorFromSet(monitorNamespaceLabel).String()
		option := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.LabelSelector = labelSelector
		})

		monitorSecretLabel := map[string]string{constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue}
		secretLabelSelector := fields.SelectorFromSet(monitorSecretLabel).String()
		secretOption := informers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.LabelSelector = secretLabelSelector
		})

		nsInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, DefaultKubeEventResyncInterval, option)
		informerFactory := informers.NewSharedInformerFactory(kubeClient, DefaultKubeEventResyncInterval)
		secretInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, DefaultKubeEventResyncInterval, secretOption)
		v1api := informerFactory.Core().V1()
		c.informers[informerKeyNamespace] = nsInformerFactory.Core().V1().Namespaces().Informer()
		c.informers[informerKeyService] = v1api.Services().Informer()
		c.informers[informerKeyServiceAccount] = v1api.ServiceAccounts().Informer()
		c.informers[informerKeyPod] = v1api.Pods().Informer()
		c.informers[informerKeyEndpoints] = v1api.Endpoints().Informer()
		c.informers[informerKeySecret] = secretInformerFactory.Core().V1().Secrets().Informer()
	}
}

// WithSMIClients sets the SMI clients for the Client
func WithSMIClients(smiTrafficSplitClient smiTrafficSplitClient.Interface, smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiTrafficAccessClient.Interface) ClientOption {
	return func(c *Client) {
		accessInformerFactory := smiAccessInformers.NewSharedInformerFactory(smiAccessClient, DefaultKubeEventResyncInterval)
		splitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, DefaultKubeEventResyncInterval)
		specInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, DefaultKubeEventResyncInterval)

		c.informers[informerKeyTCPRoute] = specInformerFactory.Specs().V1alpha4().TCPRoutes().Informer()
		c.informers[informerKeyHTTPRouteGroup] = specInformerFactory.Specs().V1alpha4().HTTPRouteGroups().Informer()
		c.informers[informerKeyTrafficTarget] = accessInformerFactory.Access().V1alpha3().TrafficTargets().Informer()
		c.informers[informerKeyTrafficSplit] = splitInformerFactory.Split().V1alpha2().TrafficSplits().Informer()
	}
}

// WithConfigClient sets the config client for the Client
func WithConfigClient(configClient configClientset.Interface) ClientOption {
	return func(c *Client) {
		c.configClient = configClient

		listOption := configInformers.WithTweakListOptions(func(opt *metav1.ListOptions) {
			opt.FieldSelector = fields.OneTermEqualSelector(metav1.ObjectNameField, c.meshConfigName).String()
		})
		meshConfiginformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, DefaultKubeEventResyncInterval, configInformers.WithNamespace(c.osmNamespace), listOption)
		mrcInformerFactory := configInformers.NewSharedInformerFactoryWithOptions(configClient, MRCResyncInterval, configInformers.WithNamespace(c.osmNamespace))
		informerFactory := configInformers.NewSharedInformerFactory(configClient, DefaultKubeEventResyncInterval)

		c.informers[informerKeyMeshConfig] = meshConfiginformerFactory.Config().V1alpha2().MeshConfigs().Informer()
		c.informers[informerKeyMeshRootCertificate] = mrcInformerFactory.Config().V1alpha2().MeshRootCertificates().Informer()
		c.informers[informerKeyExtensionService] = informerFactory.Config().V1alpha2().ExtensionServices().Informer()
	}
}

// WithMCSClient sets the MCS clients for the Client
func WithMCSClient(client mcsClient.Interface) ClientOption {
	return func(c *Client) {
		c.mcsClient = client
		mcsInformerFactory := mcsInformers.NewSharedInformerFactory(client, DefaultKubeEventResyncInterval)

		c.informers[informerKeyServiceImport] = mcsInformerFactory.Multicluster().V1alpha1().ServiceImports().Informer()
		c.informers[informerKeyServiceExport] = mcsInformerFactory.Multicluster().V1alpha1().ServiceExports().Informer()
	}
}

// WithPolicyClient sets the policy client for the Client
func WithPolicyClient(policyClient policyClientset.Interface) ClientOption {
	return func(c *Client) {
		c.policyClient = policyClient

		informerFactory := policyInformers.NewSharedInformerFactory(policyClient, DefaultKubeEventResyncInterval)

		c.informers[informerKeyEgress] = informerFactory.Policy().V1alpha1().Egresses().Informer()
		c.informers[informerKeyIngressBackend] = informerFactory.Policy().V1alpha1().IngressBackends().Informer()
		c.informers[informerKeyUpstreamTrafficSetting] = informerFactory.Policy().V1alpha1().UpstreamTrafficSettings().Informer()
		c.informers[informerKeyRetry] = informerFactory.Policy().V1alpha1().Retries().Informer()
		c.informers[informerKeyTelemetry] = informerFactory.Policy().V1alpha1().Telemetries().Informer()
	}
}

func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("Client started")
	var hasSynced []cache.InformerSynced
	var names []string

	if c.informers == nil {
		return errInitInformers
	}

	handler := c.defaultEventHandler()

	for name, informer := range c.informers {
		if informer == nil {
			continue
		}

		informer.AddEventHandler(handler)

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

// getByKey retrieves an item (based on the given index) from the store of the informer indexed by the given informerKey
func (c *Client) getByKey(informerKey informerKey, objectKey string) (interface{}, bool, error) {
	informer, ok := c.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil, false, nil
	}

	return informer.GetStore().GetByKey(objectKey)
}

// list returns the contents of the store of the informer indexed by the given informerKey
func (c *Client) list(informerKey informerKey) []interface{} {
	informer, ok := c.informers[informerKey]
	if !ok {
		// keithmattix: This is the silent failure option, but perhaps we want to return an error?
		return nil
	}

	return informer.GetStore().List()
}

// IsMonitoredNamespace returns a boolean indicating if the namespace is among the list of monitored namespaces
func (c *Client) IsMonitoredNamespace(namespace string) bool {
	_, exists, _ := c.informers[informerKeyNamespace].GetStore().GetByKey(namespace)
	return exists
}

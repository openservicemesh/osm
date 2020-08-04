package smi

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiTrafficTargetClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficTargetInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	backpressure "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	backpressureClient "github.com/openservicemesh/osm/experimental/pkg/client/clientset/versioned"
	backpressureInformers "github.com/openservicemesh/osm/experimental/pkg/client/informers/externalversions"
	"github.com/openservicemesh/osm/pkg/featureflags"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/namespace"
	"github.com/openservicemesh/osm/pkg/service"
)

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(smiKubeConfig *rest.Config, kubeClient kubernetes.Interface, osmNamespace string, namespaceController namespace.Controller, stop chan struct{}) (MeshSpec, error) {
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficTargetClientSet := smiTrafficTargetClient.NewForConfigOrDie(smiKubeConfig)

	var backpressureClientSet *backpressureClient.Clientset
	if featureflags.IsBackpressureEnabled() {
		backpressureClientSet = backpressureClient.NewForConfigOrDie(smiKubeConfig)
	}

	client := newSMIClient(
		kubeClient,
		smiTrafficSplitClientSet,
		smiTrafficSpecClientSet,
		smiTrafficTargetClientSet,
		backpressureClientSet,
		osmNamespace,
		namespaceController,
		kubernetesClientName,
	)

	err := client.run(stop)
	if err != nil {
		return client, errors.Errorf("Could not start %s client", kubernetesClientName)
	}
	return client, nil
}

func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("SMI Client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[string]cache.SharedInformer{
		"TrafficSplit":  c.informers.TrafficSplit,
		"Services":      c.informers.Services,
		"TrafficSpec":   c.informers.TrafficSpec,
		"TrafficTarget": c.informers.TrafficTarget,
	}

	if featureflags.IsBackpressureEnabled() {
		sharedInformers["Backpressure"] = c.informers.Backpressure
	}

	var names []string
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		log.Info().Msgf("Starting informer: %s", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	log.Info().Msgf("[SMI Client] Waiting for informers' cache to sync: %+v", strings.Join(names, ", "))
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	log.Info().Msgf("[SMI Client] Cache sync finished for %+v", names)
	return nil
}

// GetID implements endpoints.Provider interface and returns a string descriptor / identifier of the compute provider.
func (c *Client) GetID() string {
	return c.providerIdent
}

// GetAnnouncementsChannel returns the announcement channel for the SMI client.
func (c *Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(kubeClient kubernetes.Interface, smiTrafficSplitClient *smiTrafficSplitClient.Clientset, smiTrafficSpecClient *smiTrafficSpecClient.Clientset, smiTrafficTargetClient *smiTrafficTargetClient.Clientset, backpressureClient *backpressureClient.Clientset, osmNamespace string, namespaceController namespace.Controller, providerIdent string) *Client {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSplitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSpecInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficTargetInformerFactory := smiTrafficTargetInformers.NewSharedInformerFactory(smiTrafficTargetClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		Services:      informerFactory.Core().V1().Services().Informer(),
		TrafficSplit:  smiTrafficSplitInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		TrafficSpec:   smiTrafficSpecInformerFactory.Specs().V1alpha3().HTTPRouteGroups().Informer(),
		TrafficTarget: smiTrafficTargetInformerFactory.Access().V1alpha2().TrafficTargets().Informer(),
	}

	cacheCollection := CacheCollection{
		Services:      informerCollection.Services.GetStore(),
		TrafficSplit:  informerCollection.TrafficSplit.GetStore(),
		TrafficSpec:   informerCollection.TrafficSpec.GetStore(),
		TrafficTarget: informerCollection.TrafficTarget.GetStore(),
	}

	if featureflags.IsBackpressureEnabled() {
		backPressureInformerFactory := backpressureInformers.NewSharedInformerFactoryWithOptions(backpressureClient, k8s.DefaultKubeEventResyncInterval)
		informerCollection.Backpressure = backPressureInformerFactory.Policy().V1alpha1().Backpressures().Informer()
		cacheCollection.Backpressure = informerCollection.Backpressure.GetStore()
	}

	client := Client{
		providerIdent:       providerIdent,
		informers:           &informerCollection,
		caches:              &cacheCollection,
		cacheSynced:         make(chan interface{}),
		announcements:       make(chan interface{}),
		osmNamespace:        osmNamespace,
		namespaceController: namespaceController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return namespaceController.IsMonitoredNamespace(ns)
	}
	informerCollection.Services.AddEventHandler(k8s.GetKubernetesEventHandlers("Services", "SMI", client.announcements, shouldObserve))
	informerCollection.TrafficSplit.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficSplit", "SMI", client.announcements, shouldObserve))
	informerCollection.TrafficSpec.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficSpec", "SMI", client.announcements, shouldObserve))
	informerCollection.TrafficTarget.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficTarget", "SMI", client.announcements, shouldObserve))

	if featureflags.IsBackpressureEnabled() {
		informerCollection.Backpressure.AddEventHandler(k8s.GetKubernetesEventHandlers("Backpressure", "SMI", client.announcements, shouldObserve))
	}

	return &client
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*split.TrafficSplit {
	var trafficSplits []*split.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		trafficSplit, ok := splitIface.(*split.TrafficSplit)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for TrafficSplit in cache")
			continue
		}
		if !c.namespaceController.IsMonitoredNamespace(trafficSplit.Namespace) {
			continue
		}
		trafficSplits = append(trafficSplits, trafficSplit)
	}
	return trafficSplits
}

// ListHTTPTrafficSpecs implements mesh.Topology by returning the list of traffic specs.
func (c *Client) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	var httpTrafficSpec []*spec.HTTPRouteGroup
	for _, specIface := range c.caches.TrafficSpec.List() {
		routeGroup, ok := specIface.(*spec.HTTPRouteGroup)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for HTTPRouteGroup in cache")
			continue
		}
		if !c.namespaceController.IsMonitoredNamespace(routeGroup.Namespace) {
			continue
		}
		httpTrafficSpec = append(httpTrafficSpec, routeGroup)
	}
	return httpTrafficSpec
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *Client) ListTrafficTargets() []*target.TrafficTarget {
	var trafficTargets []*target.TrafficTarget
	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget, ok := targetIface.(*target.TrafficTarget)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for TrafficTarget in cache")
			continue
		}
		if !c.namespaceController.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}
		trafficTargets = append(trafficTargets, trafficTarget)
	}
	return trafficTargets
}

// ListBackpressures implements smi.MeshSpec and returns a list of backpressure policies.
func (c *Client) ListBackpressures() []*backpressure.Backpressure {
	var backpressureList []*backpressure.Backpressure

	if !featureflags.IsBackpressureEnabled() {
		log.Info().Msgf("Backpressure turned off!")
		return backpressureList
	}

	for _, pressureIface := range c.caches.Backpressure.List() {
		bpressure, ok := pressureIface.(*backpressure.Backpressure)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Object obtained from cache is not *Backpressure")
			continue
		}

		if !c.namespaceController.IsMonitoredNamespace(bpressure.Namespace) {
			continue
		}
		backpressureList = append(backpressureList, bpressure)
	}

	return backpressureList
}

// ListTrafficSplitServices implements mesh.MeshSpec by returning the services observed from the given compute provider
func (c *Client) ListTrafficSplitServices() []service.WeightedService {
	var services []service.WeightedService
	for _, splitIface := range c.caches.TrafficSplit.List() {
		trafficSplit, ok := splitIface.(*split.TrafficSplit)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for TrafficSplit in cache")
			continue
		}
		rootService := trafficSplit.Spec.Service
		for _, backend := range trafficSplit.Spec.Backends {
			// The TrafficSplit SMI Spec does not allow providing a namespace for the backends,
			// so we assume that the top level namespace for the TrafficSplit is the namespace
			// the backends belong to.
			meshService := service.MeshService{
				Namespace: trafficSplit.Namespace,
				Name:      backend.Service,
			}
			services = append(services, service.WeightedService{Service: meshService, Weight: backend.Weight, RootService: rootService})
		}
	}
	return services
}

// ListServiceAccounts implements mesh.MeshSpec by returning the service accounts observed from the given compute provider
func (c *Client) ListServiceAccounts() []service.K8sServiceAccount {
	var serviceAccounts []service.K8sServiceAccount
	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget, ok := targetIface.(*target.TrafficTarget)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for TrafficTarget in cache")
			continue
		}
		for _, sources := range trafficTarget.Spec.Sources {
			// Only monitor sources in namespaces OSM is observing
			if !c.namespaceController.IsMonitoredNamespace(sources.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			namespacedServiceAccount := service.K8sServiceAccount{
				Namespace: sources.Namespace,
				Name:      sources.Name,
			}
			serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
		}

		// Only monitor destination in namespaces OSM is observing
		if !c.namespaceController.IsMonitoredNamespace(trafficTarget.Spec.Destination.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		namespacedServiceAccount := service.K8sServiceAccount{
			Namespace: trafficTarget.Spec.Destination.Namespace,
			Name:      trafficTarget.Spec.Destination.Name,
		}
		serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
	}
	return serviceAccounts
}

// GetService retrieves the Kubernetes Services resource for the given MeshService
func (c *Client) GetService(svc service.MeshService) (service *corev1.Service, err error) {
	// client-go cache uses <namespace>/<name> as key
	svcIf, exists, err := c.caches.Services.GetByKey(svc.String())
	if exists && err == nil {
		svc, ok := svcIf.(*corev1.Service)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msgf("Failed type assertion for MeshService in cache")
			return nil, errInvalidObjectType
		}
		return svc, nil
	}
	return nil, err
}

// ListServices returns a list of services that are part of monitored namespaces
func (c Client) ListServices() ([]*corev1.Service, error) {
	var services []*corev1.Service

	for _, serviceInterface := range c.caches.Services.List() {
		svc, ok := serviceInterface.(*corev1.Service)
		if !ok {
			log.Error().Err(errInvalidObjectType).Msg("Failed type assertion for MeshService in cache")
			continue
		}
		if !c.namespaceController.IsMonitoredNamespace(svc.Namespace) {
			continue
		}
		services = append(services, svc)
	}
	return services, nil
}

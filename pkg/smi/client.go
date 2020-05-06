package smi

import (
	"reflect"
	"strings"

	"github.com/open-service-mesh/osm/pkg/endpoint"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha1"
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

	k8s "github.com/open-service-mesh/osm/pkg/kubernetes"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(kubeConfig *rest.Config, osmNamespace string, namespaceController namespace.Controller, stop chan struct{}) MeshSpec {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiTrafficTargetClient.NewForConfigOrDie(kubeConfig)

	client := newSMIClient(kubeClient, smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet, osmNamespace, namespaceController, kubernetesClientName)

	err := client.run(stop)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not start %s client", kubernetesClientName)
	}
	return client
}

// run executes informer collection.
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

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// GetAnnouncementsChannel returns the announcement channel for the SMI client.
func (c *Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(kubeClient *kubernetes.Clientset, smiTrafficSplitClient *smiTrafficSplitClient.Clientset, smiTrafficSpecClient *smiTrafficSpecClient.Clientset, smiTrafficTargetClient *smiTrafficTargetClient.Clientset, osmNamespace string, namespaceController namespace.Controller, providerIdent string) *Client {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSplitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSpecInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficTargetInformerFactory := smiTrafficTargetInformers.NewSharedInformerFactory(smiTrafficTargetClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		Services:      informerFactory.Core().V1().Services().Informer(),
		TrafficSplit:  smiTrafficSplitInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		TrafficSpec:   smiTrafficSpecInformerFactory.Specs().V1alpha1().HTTPRouteGroups().Informer(),
		TrafficTarget: smiTrafficTargetInformerFactory.Access().V1alpha1().TrafficTargets().Informer(),
	}

	cacheCollection := CacheCollection{
		Services:      informerCollection.Services.GetStore(),
		TrafficSplit:  informerCollection.TrafficSplit.GetStore(),
		TrafficSpec:   informerCollection.TrafficSpec.GetStore(),
		TrafficTarget: informerCollection.TrafficTarget.GetStore(),
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

	nsFilter := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return !namespaceController.IsMonitoredNamespace(ns)
	}
	informerCollection.Services.AddEventHandler(k8s.GetKubernetesEventHandlers("Services", "SMI", client.announcements, nsFilter))
	informerCollection.TrafficSplit.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficSplit", "SMI", client.announcements, nsFilter))
	informerCollection.TrafficSpec.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficSpec", "SMI", client.announcements, nsFilter))
	informerCollection.TrafficTarget.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficTarget", "SMI", client.announcements, nsFilter))

	return &client
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*split.TrafficSplit {
	var trafficSplits []*split.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*split.TrafficSplit)
		if !c.namespaceController.IsMonitoredNamespace(split.Namespace) {
			continue
		}
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListHTTPTrafficSpecs implements mesh.Topology by returning the list of traffic specs.
func (c *Client) ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup {
	var httpTrafficSpec []*spec.HTTPRouteGroup
	for _, specIface := range c.caches.TrafficSpec.List() {
		spec := specIface.(*spec.HTTPRouteGroup)
		if !c.namespaceController.IsMonitoredNamespace(spec.Namespace) {
			continue
		}
		httpTrafficSpec = append(httpTrafficSpec, spec)
	}
	return httpTrafficSpec
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *Client) ListTrafficTargets() []*target.TrafficTarget {
	var trafficTarget []*target.TrafficTarget
	for _, targetIface := range c.caches.TrafficTarget.List() {
		target := targetIface.(*target.TrafficTarget)
		if !c.namespaceController.IsMonitoredNamespace(target.Namespace) {
			continue
		}
		trafficTarget = append(trafficTarget, target)
	}
	return trafficTarget
}

// ListServices implements mesh.MeshSpec by returning the services observed from the given compute provider
func (c *Client) ListServices() []endpoint.WeightedService {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var services []endpoint.WeightedService
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*split.TrafficSplit)
		domain := split.Spec.Service
		for _, backend := range split.Spec.Backends {
			// The TrafficSplit SMI Spec does not allow providing a namespace for the backends,
			// so we assume that the top level namespace for the TrafficSplit is the namespace
			// the backends belong to.
			namespacedServiceName := endpoint.NamespacedService{
				Namespace: split.Namespace,
				Service:   backend.Service,
			}
			services = append(services, endpoint.WeightedService{ServiceName: namespacedServiceName, Weight: backend.Weight, Domain: domain})
		}
	}
	return services
}

// ListServiceAccounts implements mesh.MeshSpec by returning the service accounts observed from the given compute provider
func (c *Client) ListServiceAccounts() []endpoint.NamespacedServiceAccount {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var serviceAccounts []endpoint.NamespacedServiceAccount
	for _, targetIface := range c.caches.TrafficTarget.List() {
		target := targetIface.(*target.TrafficTarget)
		for _, sources := range target.Sources {
			// Only monitor sources in namespaces OSM is observing
			if !c.namespaceController.IsMonitoredNamespace(sources.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			namespacedServiceAccount := endpoint.NamespacedServiceAccount{
				Namespace:      sources.Namespace,
				ServiceAccount: sources.Name,
			}
			serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
		}

		destination := target.Destination
		// Only monitor destination in namespaces OSM is observing
		if !c.namespaceController.IsMonitoredNamespace(destination.Namespace) {
			// Doesn't belong to namespaces we are observing
			continue
		}
		namespacedServiceAccount := endpoint.NamespacedServiceAccount{
			Namespace:      destination.Namespace,
			ServiceAccount: destination.Name,
		}
		serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
	}
	return serviceAccounts
}

// GetService retrieves the Kubernetes Services resource for the given ServiceName.
func (c *Client) GetService(svc endpoint.ServiceName) (service *corev1.Service, exists bool, err error) {
	svcIf, exists, err := c.caches.Services.GetByKey(string(svc))
	if exists && err == nil {
		return svcIf.(*corev1.Service), exists, err
	}
	return nil, exists, err
}

package smi

import (
	"time"

	target "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha1"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha1"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	smiTrafficTargetClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficTargetInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/golang/glog"
	"github.com/open-service-mesh/osm/pkg/endpoint"
	"github.com/open-service-mesh/osm/pkg/log/level"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var resyncPeriod = 10 * time.Second

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(kubeConfig *rest.Config, osmNamespace string, namespaces []string, stop chan struct{}) MeshSpec {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiTrafficTargetClient.NewForConfigOrDie(kubeConfig)

	client := newSMIClient(kubeClient, smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet, osmNamespace, namespaces, kubernetesClientName)

	err := client.run(stop)
	if err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return client
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	glog.V(level.Info).Infoln("SMI Client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[friendlyName]cache.SharedInformer{
		"TrafficSplit":  c.informers.TrafficSplit,
		"Services":      c.informers.Services,
		"TrafficSpec":   c.informers.TrafficSpec,
		"TrafficTarget": c.informers.TrafficTarget,
	}

	var names []friendlyName
	for name, informer := range sharedInformers {
		// Depending on the use-case, some Informers from the collection may not have been initialized.
		if informer == nil {
			continue
		}
		names = append(names, name)
		glog.Info("Starting informer: ", name)
		go informer.Run(stop)
		hasSynced = append(hasSynced, informer.HasSynced)
	}

	glog.V(level.Info).Infof("[SMI Client] Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(level.Info).Infof("[SMI Client] Cache sync finished for %+v", names)
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
func newSMIClient(kubeClient *kubernetes.Clientset, smiTrafficSplitClient *smiTrafficSplitClient.Clientset, smiTrafficSpecClient *smiTrafficSpecClient.Clientset, smiTrafficTargetClient *smiTrafficTargetClient.Clientset, osmNamespace string, namespaces []string, providerIdent string) *Client {
	informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)
	smiTrafficSplitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, resyncPeriod)
	smiTrafficSpecInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, resyncPeriod)
	smiTrafficTargetInformerFactory := smiTrafficTargetInformers.NewSharedInformerFactory(smiTrafficTargetClient, resyncPeriod)

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
		providerIdent: providerIdent,
		informers:     &informerCollection,
		caches:        &cacheCollection,
		cacheSynced:   make(chan interface{}),
		announcements: make(chan interface{}),
		osmNamespace:  osmNamespace,
		namespaces:    make(map[string]struct{}),
	}
	for _, ns := range namespaces {
		client.namespaces[ns] = struct{}{}
	}

	h := handlers{client}

	resourceHandler := cache.ResourceEventHandlerFuncs{
		AddFunc:    h.addFunc,
		UpdateFunc: h.updateFunc,
		DeleteFunc: h.deleteFunc,
	}

	informerCollection.Services.AddEventHandler(resourceHandler)
	informerCollection.TrafficSplit.AddEventHandler(resourceHandler)
	informerCollection.TrafficSpec.AddEventHandler(resourceHandler)
	informerCollection.TrafficTarget.AddEventHandler(resourceHandler)

	return &client
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*split.TrafficSplit {
	var trafficSplits []*split.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*split.TrafficSplit)
		if c.IsNotObservedNamespace(split.Namespace) {
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
		if c.IsNotObservedNamespace(spec.Namespace) {
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
		if c.IsNotObservedNamespace(target.Namespace) {
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
		for _, backend := range split.Spec.Backends {
			// The TrafficSplit SMI Spec does not allow providing a namespace for the backends,
			// so we assume that the top level namespace for the TrafficSplit is the namespace
			// the backends belong to.
			namespacedServiceName := endpoint.NamespacedService{
				Namespace: split.Namespace,
				Service:   backend.Service,
			}
			services = append(services, endpoint.WeightedService{ServiceName: namespacedServiceName, Weight: backend.Weight})
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
			if c.IsNotObservedNamespace(sources.Namespace) {
				// Doesn't belong to namespaces we are observing
				glog.V(level.Trace).Infof("Namespace %q for traffic sources not in the list of observing namespaces %v, skipping.", sources.Namespace, c.namespaces)
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
		if c.IsNotObservedNamespace(destination.Namespace) {
			// Doesn't belong to namespaces we are observing
			glog.V(level.Trace).Infof("Namespace %q for traffic destination not in the list of observing namespaces %v, skipping.", destination.Namespace, c.namespaces)
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

// IsNotObservedNamespace returns true if the namespace does not belong to a non-empty list of namespaces the Client is observing
func (c Client) IsNotObservedNamespace(namespace string) bool {
	_, exists := c.namespaces[namespace]
	return len(c.namespaces) > 0 && !exists
}

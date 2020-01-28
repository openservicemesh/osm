package smi

import (
	"fmt"
	"time"

	TrafficTarget "github.com/deislabs/smi-sdk-go/pkg/apis/access/v1alpha1"
	TrafficSpec "github.com/deislabs/smi-sdk-go/pkg/apis/specs/v1alpha1"
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/deislabs/smc/pkg/endpoint"
	smiTrafficTargetClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficTargetExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClientVersion "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitExternalVersions "github.com/deislabs/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"github.com/golang/glog"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var resyncPeriod = 1 * time.Second

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(kubeConfig *rest.Config, namespaces []string, announcements chan interface{}, stop chan struct{}) MeshSpec {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)
	smiTrafficSplitClientSet := smiTrafficSplitClientVersion.NewForConfigOrDie(kubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClientVersion.NewForConfigOrDie(kubeConfig)
	smiTrafficTargetClientSet := smiTrafficTargetClientVersion.NewForConfigOrDie(kubeConfig)

	client := newSMIClient(kubeClient, smiTrafficSplitClientSet, smiTrafficSpecClientSet, smiTrafficTargetClientSet, namespaces, announcements, kubernetesClientName)
	//	client := newSMIClient(kubeClient, smiClientset, namespaces, announcements, kubernetesClientName)
	err := client.run(stop)
	if err != nil {
		glog.Fatalf("Could not start %s client: %s", kubernetesClientName, err)
	}
	return client
}

// run executes informer collection.
func (c *Client) run(stop <-chan struct{}) error {
	glog.V(1).Infoln("SMI Client started")
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

	glog.V(1).Infof("[SMI Client] Waiting informers cache sync: %+v", names)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	// Closing the cacheSynced channel signals to the rest of the system that... caches have been synced.
	close(c.cacheSynced)

	glog.V(1).Infof("[SMI Client] Cache sync finished for %+v", names)
	return nil
}

// GetID returns a string descriptor / identifier of the compute provider.
// Required by interface: EndpointsProvider
func (c *Client) GetID() string {
	return c.providerIdent
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(kubeClient *kubernetes.Clientset, smiTrafficSplitClient *smiTrafficSplitClientVersion.Clientset, smiTrafficSpecClient *smiTrafficSpecClientVersion.Clientset, smiTrafficTargetClient *smiTrafficTargetClientVersion.Clientset, namespaces []string, announcements chan interface{}, providerIdent string) *Client {
	// func newSMIClient(kubeClient *kubernetes.Clientset, smiClient *versioned.Clientset, namespaces []string, announcements chan interface{}, providerIdent string) *Client {
	var options []informers.SharedInformerOption
	var smiTrafficSplitOptions []smiTrafficSplitExternalVersions.SharedInformerOption
	var smiTrafficSpecOptions []smiTrafficSpecExternalVersions.SharedInformerOption
	var smiTrafficTargetOptions []smiTrafficTargetExternalVersions.SharedInformerOption
	for _, namespace := range namespaces {
		options = append(options, informers.WithNamespace(namespace))
		smiTrafficSplitOptions = append(smiTrafficSplitOptions, smiTrafficSplitExternalVersions.WithNamespace(namespace))
		smiTrafficSpecOptions = append(smiTrafficSpecOptions, smiTrafficSpecExternalVersions.WithNamespace(namespace))
		smiTrafficTargetOptions = append(smiTrafficTargetOptions, smiTrafficTargetExternalVersions.WithNamespace(namespace))
	}

	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, resyncPeriod, options...)
	smiTrafficSplitInformerFactory := smiTrafficSplitExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficSplitClient, resyncPeriod, smiTrafficSplitOptions...)
	smiTrafficSpecInformerFactory := smiTrafficSpecExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficSpecClient, resyncPeriod, smiTrafficSpecOptions...)
	smiTrafficTargetInformerFactory := smiTrafficTargetExternalVersions.NewSharedInformerFactoryWithOptions(smiTrafficTargetClient, resyncPeriod, smiTrafficTargetOptions...)

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
		announcements: announcements,
		cacheSynced:   make(chan interface{}),
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
func (c *Client) ListTrafficSplits() []*v1alpha2.TrafficSplit {
	var trafficSplits []*v1alpha2.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		trafficSplits = append(trafficSplits, split)
	}
	return trafficSplits
}

// ListHTTPTrafficSpecs implements mesh.Topology by returning the list of traffic specs.
func (c *Client) ListHTTPTrafficSpecs() []*TrafficSpec.HTTPRouteGroup {
	var httpTrafficSpec []*TrafficSpec.HTTPRouteGroup
	for _, specIface := range c.caches.TrafficSpec.List() {
		spec := specIface.(*TrafficSpec.HTTPRouteGroup)
		httpTrafficSpec = append(httpTrafficSpec, spec)
	}
	return httpTrafficSpec
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *Client) ListTrafficTargets() []*TrafficTarget.TrafficTarget {
	var trafficTarget []*TrafficTarget.TrafficTarget
	for _, targetIface := range c.caches.TrafficTarget.List() {
		target := targetIface.(*TrafficTarget.TrafficTarget)
		trafficTarget = append(trafficTarget, target)
	}
	return trafficTarget
}

// ListServices implements mesh.MeshSpec by returning the services observed from the given compute provider
func (c *Client) ListServices() []endpoint.ServiceName {
	// TODO(draychev): split the namespace and the service kubernetesClientName -- for non-kubernetes services we won't have namespace
	var services []endpoint.ServiceName
	for _, splitIface := range c.caches.TrafficSplit.List() {
		split := splitIface.(*v1alpha2.TrafficSplit)
		namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, split.Spec.Service)
		services = append(services, endpoint.ServiceName(namespacedServiceName))
		for _, backend := range split.Spec.Backends {
			namespacedServiceName := fmt.Sprintf("%s/%s", split.Namespace, backend.Service)
			services = append(services, endpoint.ServiceName(namespacedServiceName))
		}
	}
	return services
}

// GetService retrieves the Kubernetes Services resource for the given ServiceName.
func (c *Client) GetService(svc endpoint.ServiceName) (service *v1.Service, exists bool, err error) {
	svcIf, exists, err := c.caches.Services.GetByKey(string(svc))
	if exists && err == nil {
		return svcIf.(*v1.Service), exists, err
	}
	return nil, exists, err
}

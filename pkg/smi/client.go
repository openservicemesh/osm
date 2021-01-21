package smi

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiAccessInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/informers/externalversions"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSpecInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/informers/externalversions"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	smiTrafficSplitInformers "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/informers/externalversions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	osmPolicy "github.com/openservicemesh/osm/experimental/pkg/apis/policy/v1alpha1"
	osmPolicyClient "github.com/openservicemesh/osm/experimental/pkg/client/clientset/versioned"
	backpressureInformers "github.com/openservicemesh/osm/experimental/pkg/client/informers/externalversions"
	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/featureflags"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/service"
)

// We have a few different k8s clients. This identifies these in logs.
const kubernetesClientName = "MeshSpec"

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(smiKubeConfig *rest.Config, kubeClient kubernetes.Interface, osmNamespace string, kubeController k8s.Controller, stop chan struct{}) (MeshSpec, error) {
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficTargetClientSet := smiAccessClient.NewForConfigOrDie(smiKubeConfig)

	var backpressureClientSet *osmPolicyClient.Clientset
	if featureflags.IsBackpressureEnabled() {
		backpressureClientSet = osmPolicyClient.NewForConfigOrDie(smiKubeConfig)
	}

	client, err := newSMIClient(
		kubeClient,
		smiTrafficSplitClientSet,
		smiTrafficSpecClientSet,
		smiTrafficTargetClientSet,
		backpressureClientSet,
		osmNamespace,
		kubeController,
		kubernetesClientName,
		stop,
	)

	return client, err
}

func (c *Client) run(stop <-chan struct{}) error {
	log.Info().Msg("SMI Client started")
	var hasSynced []cache.InformerSynced

	if c.informers == nil {
		return errInitInformers
	}

	sharedInformers := map[string]cache.SharedInformer{
		"TrafficSplit":   c.informers.TrafficSplit,
		"HTTPRouteGroup": c.informers.HTTPRouteGroup,
		"TCPRoute":       c.informers.TCPRoute,
		"TrafficTarget":  c.informers.TrafficTarget,
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

// GetAnnouncementsChannel returns the announcement channel for the SMI client.
func (c *Client) GetAnnouncementsChannel() <-chan a.Announcement {
	return c.announcements
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(kubeClient kubernetes.Interface, smiTrafficSplitClient smiTrafficSplitClient.Interface, smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiAccessClient.Interface, backpressureClient osmPolicyClient.Interface, osmNamespace string, kubeController k8s.Controller, providerIdent string, stop chan struct{}) (*Client, error) {
	smiTrafficSplitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSpecInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficTargetInformerFactory := smiAccessInformers.NewSharedInformerFactory(smiAccessClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := InformerCollection{
		TrafficSplit:   smiTrafficSplitInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		HTTPRouteGroup: smiTrafficSpecInformerFactory.Specs().V1alpha4().HTTPRouteGroups().Informer(),
		TCPRoute:       smiTrafficSpecInformerFactory.Specs().V1alpha4().TCPRoutes().Informer(),
		TrafficTarget:  smiTrafficTargetInformerFactory.Access().V1alpha3().TrafficTargets().Informer(),
	}

	cacheCollection := CacheCollection{
		TrafficSplit:   informerCollection.TrafficSplit.GetStore(),
		HTTPRouteGroup: informerCollection.HTTPRouteGroup.GetStore(),
		TCPRoute:       informerCollection.TCPRoute.GetStore(),
		TrafficTarget:  informerCollection.TrafficTarget.GetStore(),
	}

	if featureflags.IsBackpressureEnabled() {
		backPressureInformerFactory := backpressureInformers.NewSharedInformerFactoryWithOptions(backpressureClient, k8s.DefaultKubeEventResyncInterval)
		informerCollection.Backpressure = backPressureInformerFactory.Policy().V1alpha1().Backpressures().Informer()
		cacheCollection.Backpressure = informerCollection.Backpressure.GetStore()
	}

	client := Client{
		providerIdent:  providerIdent,
		informers:      &informerCollection,
		caches:         &cacheCollection,
		cacheSynced:    make(chan interface{}),
		announcements:  make(chan a.Announcement),
		osmNamespace:   osmNamespace,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		ns := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
		return kubeController.IsMonitoredNamespace(ns)
	}

	splitEventTypes := k8s.EventTypes{
		Add:    a.TrafficSplitAdded,
		Update: a.TrafficSplitUpdated,
		Delete: a.TrafficSplitDeleted,
	}
	informerCollection.TrafficSplit.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficSplit", "SMI", shouldObserve, splitEventTypes))

	routeGroupEventTypes := k8s.EventTypes{
		Add:    a.RouteGroupAdded,
		Update: a.RouteGroupUpdated,
		Delete: a.RouteGroupDeleted,
	}
	informerCollection.HTTPRouteGroup.AddEventHandler(k8s.GetKubernetesEventHandlers("HTTPRouteGroup", "SMI", shouldObserve, routeGroupEventTypes))

	tcpRouteEventTypes := k8s.EventTypes{
		Add:    a.TCPRouteAdded,
		Update: a.TCPRouteUpdated,
		Delete: a.TCPRouteDeleted,
	}
	informerCollection.TCPRoute.AddEventHandler(k8s.GetKubernetesEventHandlers("TCPRoute", "SMI", shouldObserve, tcpRouteEventTypes))

	trafficTargetEventTypes := k8s.EventTypes{
		Add:    a.TrafficTargetAdded,
		Update: a.TrafficTargetUpdated,
		Delete: a.TrafficTargetDeleted,
	}
	informerCollection.TrafficTarget.AddEventHandler(k8s.GetKubernetesEventHandlers("TrafficTarget", "SMI", shouldObserve, trafficTargetEventTypes))

	if featureflags.IsBackpressureEnabled() {
		backpressureEventTypes := k8s.EventTypes{
			Add:    a.BackpressureAdded,
			Update: a.BackpressureUpdated,
			Delete: a.BackpressureDeleted,
		}
		informerCollection.Backpressure.AddEventHandler(k8s.GetKubernetesEventHandlers("Backpressure", "SMI", shouldObserve, backpressureEventTypes))
	}

	err := client.run(stop)
	if err != nil {
		return &client, errors.Errorf("Could not start %s client: %s", kubernetesClientName, err)
	}

	return &client, err
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits() []*smiSplit.TrafficSplit {
	var trafficSplits []*smiSplit.TrafficSplit
	for _, splitIface := range c.caches.TrafficSplit.List() {
		trafficSplit := splitIface.(*smiSplit.TrafficSplit)

		if !c.kubeController.IsMonitoredNamespace(trafficSplit.Namespace) {
			continue
		}
		trafficSplits = append(trafficSplits, trafficSplit)
	}
	return trafficSplits
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (c *Client) ListHTTPTrafficSpecs() []*smiSpecs.HTTPRouteGroup {
	var httpTrafficSpec []*smiSpecs.HTTPRouteGroup
	for _, specIface := range c.caches.HTTPRouteGroup.List() {
		routeGroup := specIface.(*smiSpecs.HTTPRouteGroup)

		if !c.kubeController.IsMonitoredNamespace(routeGroup.Namespace) {
			continue
		}
		httpTrafficSpec = append(httpTrafficSpec, routeGroup)
	}
	return httpTrafficSpec
}

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (c *Client) ListTCPTrafficSpecs() []*smiSpecs.TCPRoute {
	var tcpRouteSpec []*smiSpecs.TCPRoute
	for _, specIface := range c.caches.TCPRoute.List() {
		tcpRoute := specIface.(*smiSpecs.TCPRoute)

		if !c.kubeController.IsMonitoredNamespace(tcpRoute.Namespace) {
			continue
		}
		tcpRouteSpec = append(tcpRouteSpec, tcpRoute)
	}
	return tcpRouteSpec
}

// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
func (c *Client) GetTCPRoute(namespacedName string) *smiSpecs.TCPRoute {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.caches.TCPRoute.GetByKey(namespacedName)
	if exists && err == nil {
		route := routeIf.(*smiSpecs.TCPRoute)
		return route
	}
	return nil
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *Client) ListTrafficTargets() []*smiAccess.TrafficTarget {
	var trafficTargets []*smiAccess.TrafficTarget
	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		if !c.kubeController.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}
		trafficTargets = append(trafficTargets, trafficTarget)
	}
	return trafficTargets
}

// GetBackpressurePolicy gets the Backpressure policy corresponding to the MeshService
func (c *Client) GetBackpressurePolicy(svc service.MeshService) *osmPolicy.Backpressure {
	if !featureflags.IsBackpressureEnabled() {
		log.Debug().Msgf("Backpressure turned off!")
		return nil
	}

	for _, iface := range c.caches.Backpressure.List() {
		backpressure := iface.(*osmPolicy.Backpressure)

		if !c.kubeController.IsMonitoredNamespace(backpressure.Namespace) {
			continue
		}

		app, ok := backpressure.Labels["app"]
		if !ok {
			continue
		}

		if svc.Namespace == backpressure.Namespace && svc.Name == app {
			return backpressure
		}
	}

	return nil
}

// ListTrafficSplitServices implements mesh.MeshSpec by returning the services observed from the given compute provider
func (c *Client) ListTrafficSplitServices() []service.WeightedService {
	var services []service.WeightedService
	for _, splitIface := range c.caches.TrafficSplit.List() {
		trafficSplit := splitIface.(*smiSplit.TrafficSplit)
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

// ListServiceAccounts lists ServiceAccounts specified in SMI TrafficTarget resources
func (c *Client) ListServiceAccounts() []service.K8sServiceAccount {
	var serviceAccounts []service.K8sServiceAccount
	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		for _, sources := range trafficTarget.Spec.Sources {
			// Only monitor sources in namespaces OSM is observing
			if !c.kubeController.IsMonitoredNamespace(sources.Namespace) {
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
		if !c.kubeController.IsMonitoredNamespace(trafficTarget.Spec.Destination.Namespace) {
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

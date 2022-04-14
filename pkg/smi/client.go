package smi

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	// ServiceAccountKind is the kind specified for the destination and sources in an SMI TrafficTarget policy
	ServiceAccountKind = "ServiceAccount"

	// TCPRouteKind is the kind specified for the TCP route rules in an SMI Traffictarget policy
	TCPRouteKind = "TCPRoute"

	// HTTPRouteGroupKind is the kind specified for the HTTP route rules in an SMI Traffictarget policy
	HTTPRouteGroupKind = "HTTPRouteGroup"

	// We have a few different k8s clients. This identifies these in logs.
	kubernetesClientName = "MeshSpec"
)

// NewMeshSpecClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewMeshSpecClient(smiKubeConfig *rest.Config, osmNamespace string, kubeController k8s.Controller,
	stop chan struct{}, msgBroker *messaging.Broker) (MeshSpec, error) {
	smiTrafficSplitClientSet := smiTrafficSplitClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficSpecClientSet := smiTrafficSpecClient.NewForConfigOrDie(smiKubeConfig)
	smiTrafficTargetClientSet := smiAccessClient.NewForConfigOrDie(smiKubeConfig)

	client, err := newSMIClient(
		smiTrafficSplitClientSet,
		smiTrafficSpecClientSet,
		smiTrafficTargetClientSet,
		osmNamespace,
		kubeController,
		kubernetesClientName,
		stop,
		msgBroker,
	)

	return client, err
}

func (c *client) run(stop <-chan struct{}) error {
	log.Info().Msg("SMI client started")
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

	log.Info().Msgf("Waiting for informers %v caches to sync", names)
	if !cache.WaitForCacheSync(stop, hasSynced...) {
		return errSyncingCaches
	}

	log.Info().Msgf("Cache sync finished for informers %v", names)
	return nil
}

// newClient creates a provider based on a Kubernetes client instance.
func newSMIClient(smiTrafficSplitClient smiTrafficSplitClient.Interface,
	smiTrafficSpecClient smiTrafficSpecClient.Interface, smiAccessClient smiAccessClient.Interface,
	osmNamespace string, kubeController k8s.Controller, providerIdent string, stop chan struct{},
	msgBroker *messaging.Broker) (*client, error) {
	smiTrafficSplitInformerFactory := smiTrafficSplitInformers.NewSharedInformerFactory(smiTrafficSplitClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficSpecInformerFactory := smiTrafficSpecInformers.NewSharedInformerFactory(smiTrafficSpecClient, k8s.DefaultKubeEventResyncInterval)
	smiTrafficTargetInformerFactory := smiAccessInformers.NewSharedInformerFactory(smiAccessClient, k8s.DefaultKubeEventResyncInterval)

	informerCollection := informerCollection{
		TrafficSplit:   smiTrafficSplitInformerFactory.Split().V1alpha2().TrafficSplits().Informer(),
		HTTPRouteGroup: smiTrafficSpecInformerFactory.Specs().V1alpha4().HTTPRouteGroups().Informer(),
		TCPRoute:       smiTrafficSpecInformerFactory.Specs().V1alpha4().TCPRoutes().Informer(),
		TrafficTarget:  smiTrafficTargetInformerFactory.Access().V1alpha3().TrafficTargets().Informer(),
	}

	cacheCollection := cacheCollection{
		TrafficSplit:   informerCollection.TrafficSplit.GetStore(),
		HTTPRouteGroup: informerCollection.HTTPRouteGroup.GetStore(),
		TCPRoute:       informerCollection.TCPRoute.GetStore(),
		TrafficTarget:  informerCollection.TrafficTarget.GetStore(),
	}

	client := client{
		providerIdent:  providerIdent,
		informers:      &informerCollection,
		caches:         &cacheCollection,
		osmNamespace:   osmNamespace,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return kubeController.IsMonitoredNamespace(object.GetNamespace())
	}
	splitEventTypes := k8s.EventTypes{
		Add:    a.TrafficSplitAdded,
		Update: a.TrafficSplitUpdated,
		Delete: a.TrafficSplitDeleted,
	}
	informerCollection.TrafficSplit.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, splitEventTypes, msgBroker))

	routeGroupEventTypes := k8s.EventTypes{
		Add:    a.RouteGroupAdded,
		Update: a.RouteGroupUpdated,
		Delete: a.RouteGroupDeleted,
	}
	informerCollection.HTTPRouteGroup.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, routeGroupEventTypes, msgBroker))

	tcpRouteEventTypes := k8s.EventTypes{
		Add:    a.TCPRouteAdded,
		Update: a.TCPRouteUpdated,
		Delete: a.TCPRouteDeleted,
	}
	informerCollection.TCPRoute.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, tcpRouteEventTypes, msgBroker))

	trafficTargetEventTypes := k8s.EventTypes{
		Add:    a.TrafficTargetAdded,
		Update: a.TrafficTargetUpdated,
		Delete: a.TrafficTargetDeleted,
	}
	informerCollection.TrafficTarget.AddEventHandler(k8s.GetEventHandlerFuncs(shouldObserve, trafficTargetEventTypes, msgBroker))

	err := client.run(stop)
	if err != nil {
		return &client, errors.Errorf("Could not start %s client: %s", kubernetesClientName, err)
	}

	return &client, err
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *client) ListTrafficSplits(options ...TrafficSplitListOption) []*smiSplit.TrafficSplit {
	var trafficSplits []*smiSplit.TrafficSplit

	for _, splitIface := range c.caches.TrafficSplit.List() {
		trafficSplit := splitIface.(*smiSplit.TrafficSplit)

		if !c.kubeController.IsMonitoredNamespace(trafficSplit.Namespace) {
			continue
		}

		if filteredSplit := filterTrafficSplit(trafficSplit, options...); filteredSplit != nil {
			trafficSplits = append(trafficSplits, filteredSplit)
		}
	}
	return trafficSplits
}

// filterTrafficSplit applies the given TrafficSplitListOption filter on the given TrafficSplit object
func filterTrafficSplit(trafficSplit *smiSplit.TrafficSplit, options ...TrafficSplitListOption) *smiSplit.TrafficSplit {
	if trafficSplit == nil {
		return nil
	}

	o := &TrafficSplitListOpt{}
	for _, opt := range options {
		opt(o)
	}

	// If apex service filter option is set, ignore traffic splits whose apex service does not match
	if o.ApexService.Name != "" && (o.ApexService.Namespace != trafficSplit.Namespace ||
		o.ApexService.Name != k8s.GetServiceFromHostname(trafficSplit.Spec.Service)) {
		return nil
	}

	// If backend service filter option is set, ignore traffic splits whose backend service does not match
	if o.BackendService.Name != "" {
		if trafficSplit.Namespace != o.BackendService.Namespace {
			return nil
		}

		backendFound := false
		for _, backend := range trafficSplit.Spec.Backends {
			if backend.Service == o.BackendService.Name {
				backendFound = true
				break
			}
		}
		if !backendFound {
			return nil
		}
	}

	return trafficSplit
}

// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
func (c *client) ListHTTPTrafficSpecs() []*smiSpecs.HTTPRouteGroup {
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

// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
func (c *client) GetHTTPRouteGroup(namespacedName string) *smiSpecs.HTTPRouteGroup {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.caches.HTTPRouteGroup.GetByKey(namespacedName)
	if exists && err == nil {
		route := routeIf.(*smiSpecs.HTTPRouteGroup)
		if !c.kubeController.IsMonitoredNamespace(route.Namespace) {
			log.Warn().Msgf("HTTPRouteGroup %s found, but belongs to a namespace that is not monitored, ignoring it", namespacedName)
			return nil
		}
		return route
	}
	return nil
}

// ListTCPTrafficSpecs lists SMI TCPRoute resources
func (c *client) ListTCPTrafficSpecs() []*smiSpecs.TCPRoute {
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
func (c *client) GetTCPRoute(namespacedName string) *smiSpecs.TCPRoute {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.caches.TCPRoute.GetByKey(namespacedName)
	if exists && err == nil {
		route := routeIf.(*smiSpecs.TCPRoute)
		if !c.kubeController.IsMonitoredNamespace(route.Namespace) {
			log.Warn().Msgf("TCPRoute %s found, but belongs to a namespace that is not monitored, ignoring it", namespacedName)
			return nil
		}
		return route
	}
	return nil
}

// ListTrafficTargets implements mesh.Topology by returning the list of traffic targets.
func (c *client) ListTrafficTargets(options ...TrafficTargetListOption) []*smiAccess.TrafficTarget {
	var trafficTargets []*smiAccess.TrafficTarget

	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		if !c.kubeController.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}

		if !isValidTrafficTarget(trafficTarget) {
			continue
		}

		// Filter TrafficTarget based on the given options
		if filteredTrafficTarget := filterTrafficTarget(trafficTarget, options...); filteredTrafficTarget != nil {
			trafficTargets = append(trafficTargets, trafficTarget)
		}
	}
	return trafficTargets
}

func isValidTrafficTarget(trafficTarget *smiAccess.TrafficTarget) bool {
	// destination namespace must be same as traffic target namespace
	if trafficTarget.Namespace != trafficTarget.Spec.Destination.Namespace {
		return false
	}

	if !hasValidRules(trafficTarget.Spec.Rules) {
		return false
	}

	return true
}

// hasValidRules checks if the given SMI TrafficTarget object has valid rules
func hasValidRules(rules []smiAccess.TrafficTargetRule) bool {
	if len(rules) == 0 {
		return false
	}
	for _, rule := range rules {
		switch rule.Kind {
		case HTTPRouteGroupKind, TCPRouteKind:
			// valid Kind for rules

		default:
			log.Error().Msgf("Invalid Kind for rule %s in TrafficTarget policy %s", rule.Name, rule.Kind)
			return false
		}
	}
	return true
}

func filterTrafficTarget(trafficTarget *smiAccess.TrafficTarget, options ...TrafficTargetListOption) *smiAccess.TrafficTarget {
	if trafficTarget == nil {
		return nil
	}

	o := &TrafficTargetListOpt{}
	for _, opt := range options {
		opt(o)
	}

	if o.Destination.Name != "" && (o.Destination.Namespace != trafficTarget.Spec.Destination.Namespace ||
		o.Destination.Name != trafficTarget.Spec.Destination.Name) {
		return nil
	}

	return trafficTarget
}

// ListServiceAccounts lists ServiceAccounts specified in SMI TrafficTarget resources
func (c *client) ListServiceAccounts() []identity.K8sServiceAccount {
	var serviceAccounts []identity.K8sServiceAccount
	for _, targetIface := range c.caches.TrafficTarget.List() {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		if !c.kubeController.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}

		if !isValidTrafficTarget(trafficTarget) {
			continue
		}

		for _, sources := range trafficTarget.Spec.Sources {
			// Only monitor sources in namespaces OSM is observing
			if !c.kubeController.IsMonitoredNamespace(sources.Namespace) {
				// Doesn't belong to namespaces we are observing
				continue
			}
			namespacedServiceAccount := identity.K8sServiceAccount{
				Namespace: sources.Namespace,
				Name:      sources.Name,
			}
			serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
		}

		namespacedServiceAccount := identity.K8sServiceAccount{
			Namespace: trafficTarget.Spec.Destination.Namespace,
			Name:      trafficTarget.Spec.Destination.Name,
		}
		serviceAccounts = append(serviceAccounts, namespacedServiceAccount)
	}
	return serviceAccounts
}

// GetSmiClientVersionHTTPHandler returns an http handler that returns supported smi version information
func GetSmiClientVersionHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		versionInfo := map[string]string{
			"TrafficTarget":  smiAccess.SchemeGroupVersion.String(),
			"HTTPRouteGroup": smiSpecs.SchemeGroupVersion.String(),
			"TCPRoute":       smiSpecs.SchemeGroupVersion.String(),
			"TrafficSplit":   smiSplit.SchemeGroupVersion.String(),
		}

		if jsonVersionInfo, err := json.Marshal(versionInfo); err != nil {
			log.Error().Err(err).Msgf("Error marshaling version info struct: %+v", versionInfo)
		} else {
			_, _ = fmt.Fprint(w, string(jsonVersionInfo))
		}
	})
}

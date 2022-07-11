package smi

import (
	"encoding/json"
	"fmt"
	"net/http"

	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
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

// NewSMIClient implements mesh.MeshSpec and creates the Kubernetes client, which retrieves SMI specific CRDs.
func NewSMIClient(informerCollection *informers.InformerCollection, osmNamespace string, kubeController k8s.Controller, msgBroker *messaging.Broker) *Client {
	client := Client{
		providerIdent:  kubernetesClientName,
		informers:      informerCollection,
		osmNamespace:   osmNamespace,
		kubeController: kubeController,
	}

	shouldObserve := func(obj interface{}) bool {
		object, ok := obj.(metav1.Object)
		if !ok {
			return false
		}
		return informerCollection.IsMonitoredNamespace(object.GetNamespace())
	}
	splitEventTypes := k8s.EventTypes{
		Add:    a.TrafficSplitAdded,
		Update: a.TrafficSplitUpdated,
		Delete: a.TrafficSplitDeleted,
	}
	informerCollection.AddEventHandler(informers.InformerKeyTrafficSplit, k8s.GetEventHandlerFuncs(shouldObserve, splitEventTypes, msgBroker))

	routeGroupEventTypes := k8s.EventTypes{
		Add:    a.RouteGroupAdded,
		Update: a.RouteGroupUpdated,
		Delete: a.RouteGroupDeleted,
	}
	informerCollection.AddEventHandler(informers.InformerKeyHTTPRouteGroup, k8s.GetEventHandlerFuncs(shouldObserve, routeGroupEventTypes, msgBroker))

	tcpRouteEventTypes := k8s.EventTypes{
		Add:    a.TCPRouteAdded,
		Update: a.TCPRouteUpdated,
		Delete: a.TCPRouteDeleted,
	}
	informerCollection.AddEventHandler(informers.InformerKeyTCPRoute, k8s.GetEventHandlerFuncs(shouldObserve, tcpRouteEventTypes, msgBroker))

	trafficTargetEventTypes := k8s.EventTypes{
		Add:    a.TrafficTargetAdded,
		Update: a.TrafficTargetUpdated,
		Delete: a.TrafficTargetDeleted,
	}
	informerCollection.AddEventHandler(informers.InformerKeyTrafficTarget, k8s.GetEventHandlerFuncs(shouldObserve, trafficTargetEventTypes, msgBroker))

	return &client
}

// ListTrafficSplits implements mesh.MeshSpec by returning the list of traffic splits.
func (c *Client) ListTrafficSplits(options ...TrafficSplitListOption) []*smiSplit.TrafficSplit {
	var trafficSplits []*smiSplit.TrafficSplit

	for _, splitIface := range c.informers.List(informers.InformerKeyTrafficSplit) {
		trafficSplit := splitIface.(*smiSplit.TrafficSplit)

		if !c.kubeController.IsMonitoredNamespace(trafficSplit.Namespace) {
			continue
		}

		options = append(options, WithKubeController(c.kubeController))

		if filteredSplit := FilterTrafficSplit(trafficSplit, options...); filteredSplit != nil {
			trafficSplits = append(trafficSplits, filteredSplit)
		}
	}
	return trafficSplits
}

// FilterTrafficSplit applies the given TrafficSplitListOption filter on the given TrafficSplit object
func FilterTrafficSplit(trafficSplit *smiSplit.TrafficSplit, options ...TrafficSplitListOption) *smiSplit.TrafficSplit {
	if trafficSplit == nil {
		return nil
	}

	o := &TrafficSplitListOpt{}
	for _, opt := range options {
		opt(o)
	}

	// If apex service filter option is set, ignore traffic splits whose apex service does not match
	if o.ApexService.Name != "" && (o.ApexService.Namespace != trafficSplit.Namespace ||
		o.ApexService.Name != k8s.GetServiceFromHostname(o.KubeController, trafficSplit.Spec.Service)) {
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
func (c *Client) ListHTTPTrafficSpecs() []*smiSpecs.HTTPRouteGroup {
	var httpTrafficSpec []*smiSpecs.HTTPRouteGroup
	for _, specIface := range c.informers.List(informers.InformerKeyHTTPRouteGroup) {
		routeGroup := specIface.(*smiSpecs.HTTPRouteGroup)

		if !c.kubeController.IsMonitoredNamespace(routeGroup.Namespace) {
			continue
		}
		httpTrafficSpec = append(httpTrafficSpec, routeGroup)
	}
	return httpTrafficSpec
}

// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
func (c *Client) GetHTTPRouteGroup(namespacedName string) *smiSpecs.HTTPRouteGroup {
	// client-go cache uses <namespace>/<name> as key
	routeIf, exists, err := c.informers.GetByKey(informers.InformerKeyHTTPRouteGroup, namespacedName)
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
func (c *Client) ListTCPTrafficSpecs() []*smiSpecs.TCPRoute {
	var tcpRouteSpec []*smiSpecs.TCPRoute
	for _, specIface := range c.informers.List(informers.InformerKeyTCPRoute) {
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
	routeIf, exists, err := c.informers.GetByKey(informers.InformerKeyTCPRoute, namespacedName)
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
func (c *Client) ListTrafficTargets(options ...TrafficTargetListOption) []*smiAccess.TrafficTarget {
	var trafficTargets []*smiAccess.TrafficTarget

	for _, targetIface := range c.informers.List(informers.InformerKeyTrafficTarget) {
		trafficTarget := targetIface.(*smiAccess.TrafficTarget)

		if !c.kubeController.IsMonitoredNamespace(trafficTarget.Namespace) {
			continue
		}

		if !isValidTrafficTarget(trafficTarget) {
			continue
		}

		// Filter TrafficTarget based on the given options
		if filteredTrafficTarget := FilterTrafficTarget(trafficTarget, options...); filteredTrafficTarget != nil {
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

// FilterTrafficTarget applies the given TrafficTargetListOption filter on the given TrafficTarget object
func FilterTrafficTarget(trafficTarget *smiAccess.TrafficTarget, options ...TrafficTargetListOption) *smiAccess.TrafficTarget {
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
func (c *Client) ListServiceAccounts() []identity.K8sServiceAccount {
	var serviceAccounts []identity.K8sServiceAccount
	for _, targetIface := range c.informers.List(informers.InformerKeyTrafficTarget) {
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

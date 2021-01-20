package framework

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSpecs "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	smiTrafficAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	smiTrafficSpecClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/specs/clientset/versioned"
	smiTrafficSplitClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// smiClients Stores various SMI clients
type smiClients struct {
	SpecClient   *smiTrafficSpecClient.Clientset
	AccessClient *smiTrafficAccessClient.Clientset
	SplitClient  *smiTrafficSplitClient.Clientset
}

// InitSMIClients initializes SMI clients on OsmTestData structure
func (td *OsmTestData) InitSMIClients() error {
	td.SmiClients = &smiClients{}
	var err error

	td.SmiClients.SpecClient, err = smiTrafficSpecClient.NewForConfig(td.RestConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create traffic spec client")
	}

	td.SmiClients.AccessClient, err = smiTrafficAccessClient.NewForConfig(td.RestConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create traffic access client")
	}

	td.SmiClients.SplitClient, err = smiTrafficSplitClient.NewForConfig(td.RestConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create traffic split client")
	}

	return nil
}

// CreateHTTPRouteGroup Creates an SMI HTTPRouteGroup
func (td *OsmTestData) CreateHTTPRouteGroup(ns string, rg smiSpecs.HTTPRouteGroup) (*smiSpecs.HTTPRouteGroup, error) {
	hrg, err := td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(ns).Create(context.Background(), &rg, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTPRouteGroup")
	}
	return hrg, nil
}

// CreateTCPRoute Creates an SMI TCPRoute
func (td *OsmTestData) CreateTCPRoute(ns string, route smiSpecs.TCPRoute) (*smiSpecs.TCPRoute, error) {
	hrg, err := td.SmiClients.SpecClient.SpecsV1alpha4().TCPRoutes(ns).Create(context.Background(), &route, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TCPRoute")
	}
	return hrg, nil
}

// CreateTrafficTarget Creates an SMI TrafficTarget
func (td *OsmTestData) CreateTrafficTarget(ns string, tar smiAccess.TrafficTarget) (*smiAccess.TrafficTarget, error) {
	tt, err := td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(ns).Create(context.Background(), &tar, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TrafficTarget")
	}
	return tt, nil
}

// CreateTrafficSplit Creates an SMI TrafficSplit
func (td *OsmTestData) CreateTrafficSplit(ns string, tar smiSplit.TrafficSplit) (*smiSplit.TrafficSplit, error) {
	tt, err := td.SmiClients.SplitClient.SplitV1alpha2().TrafficSplits(ns).Create(context.Background(), &tar, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TrafficSplit")
	}
	return tt, nil
}

// SimpleAllowPolicy is a simplified struct to later get basic SMI allow policy
type SimpleAllowPolicy struct {
	RouteGroupName string

	TrafficTargetName string

	SourceSVCAccountName string
	SourceNamespace      string

	DestinationSvcAccountName string
	DestinationNamespace      string
}

// CreateSimpleAllowPolicy returns basic allow policy from source to destination, on a HTTP all-wildcard fashion
func (td *OsmTestData) CreateSimpleAllowPolicy(def SimpleAllowPolicy) (smiSpecs.HTTPRouteGroup, smiAccess.TrafficTarget) {
	routeGroup := smiSpecs.HTTPRouteGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.RouteGroupName,
		},
		Spec: smiSpecs.HTTPRouteGroupSpec{
			Matches: []smiSpecs.HTTPMatch{
				{
					Name:      "all",
					PathRegex: ".*",
					Methods:   []string{"*"},
				},
			},
		},
	}

	trafficTarget := smiAccess.TrafficTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.TrafficTargetName,
		},
		Spec: smiAccess.TrafficTargetSpec{
			Sources: []smiAccess.IdentityBindingSubject{
				{
					Kind:      "ServiceAccount",
					Name:      def.SourceSVCAccountName,
					Namespace: def.SourceNamespace,
				},
			},
			Destination: smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      def.DestinationSvcAccountName,
				Namespace: def.DestinationNamespace,
			},
			Rules: []smiAccess.TrafficTargetRule{
				{
					Kind: "HTTPRouteGroup",
					Name: def.RouteGroupName,
					Matches: []string{
						"all",
					},
				},
			},
		},
	}

	return routeGroup, trafficTarget
}

// CreateSimpleTCPAllowPolicy returns an allow policy to allow all TCP traffic from source to destination
func (td *OsmTestData) CreateSimpleTCPAllowPolicy(def SimpleAllowPolicy, port int) (smiSpecs.TCPRoute, smiAccess.TrafficTarget) {
	tcpRoute := smiSpecs.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.RouteGroupName,
		},
		Spec: smiSpecs.TCPRouteSpec{
			Matches: smiSpecs.TCPMatch{
				Name:  strconv.Itoa(port),
				Ports: []int{port},
			},
		},
	}

	trafficTarget := smiAccess.TrafficTarget{
		ObjectMeta: metav1.ObjectMeta{
			Name: def.TrafficTargetName,
		},
		Spec: smiAccess.TrafficTargetSpec{
			Sources: []smiAccess.IdentityBindingSubject{
				{
					Kind:      "ServiceAccount",
					Name:      def.SourceSVCAccountName,
					Namespace: def.SourceNamespace,
				},
			},
			Destination: smiAccess.IdentityBindingSubject{
				Kind:      "ServiceAccount",
				Name:      def.DestinationSvcAccountName,
				Namespace: def.DestinationNamespace,
			},
			Rules: []smiAccess.TrafficTargetRule{
				{
					Kind: "TCPRoute",
					Name: def.RouteGroupName,
				},
			},
		},
	}

	return tcpRoute, trafficTarget
}

// TrafficSplitBackend is a simple define to refer to a TrafficSplit backend
type TrafficSplitBackend struct {
	Name   string
	Weight int
}

// TrafficSplitDef is a simplified struct to get a TrafficSplit typed definition
type TrafficSplitDef struct {
	Name      string
	Namespace string

	TrafficSplitServiceName string
	Backends                []TrafficSplitBackend
}

// CreateSimpleTrafficSplit Creates an SMI TrafficTarget
func (td *OsmTestData) CreateSimpleTrafficSplit(def TrafficSplitDef) (smiSplit.TrafficSplit, error) {
	ts := smiSplit.TrafficSplit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      def.Name,
			Namespace: def.Namespace,
		},
		Spec: smiSplit.TrafficSplitSpec{
			Service: def.TrafficSplitServiceName,
		},
	}

	if def.Backends != nil && len(def.Backends) > 0 {
		ts.Spec.Backends = []smiSplit.TrafficSplitBackend{}

		for _, b := range def.Backends {
			ts.Spec.Backends = append(ts.Spec.Backends, smiSplit.TrafficSplitBackend{
				Service: b.Name,
				Weight:  b.Weight,
			})
		}
	}

	return ts, nil
}

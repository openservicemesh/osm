package specs

import (
	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	spec "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha4"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/service"
)

// MeshSpec is an interface declaring functions, which provide the specs for a service mesh declared with SMI.
type MeshSpec interface {
	// ListTrafficSplits lists SMI TrafficSplit resources
	ListTrafficSplits(...TrafficSplitListOption) []*split.TrafficSplit

	// ListServiceAccounts lists ServiceAccount resources specified in SMI TrafficTarget resources
	ListServiceAccounts() []identity.K8sServiceAccount

	// ListHTTPTrafficSpecs lists SMI HTTPRouteGroup resources
	ListHTTPTrafficSpecs() []*spec.HTTPRouteGroup

	// GetHTTPRouteGroup returns an SMI HTTPRouteGroup resource given its name of the form <namespace>/<name>
	GetHTTPRouteGroup(string) *spec.HTTPRouteGroup

	// ListTCPTrafficSpecs lists SMI TCPRoute resources
	ListTCPTrafficSpecs() []*spec.TCPRoute

	// GetTCPRoute returns an SMI TCPRoute resource given its name of the form <namespace>/<name>
	GetTCPRoute(string) *spec.TCPRoute

	// ListTrafficTargets lists SMI TrafficTarget resources. An optional filter can be applied to filter the
	// returned list
	ListTrafficTargets(...TrafficTargetListOption) []*access.TrafficTarget
}

// TrafficTargetListOpt specifies the options used to filter TrafficTarget objects as a part of its lister
type TrafficTargetListOpt struct {
	Destination identity.K8sServiceAccount
}

// TrafficTargetListOption is a function type that implements filters on TrafficTarget lister
type TrafficTargetListOption func(o *TrafficTargetListOpt)

// TrafficSplitListOpt specifies the options used to filter TrafficSplit objects as a part of its lister
type TrafficSplitListOpt struct {
	ApexService    service.MeshService
	BackendService service.MeshService
}

// TrafficSplitListOption is a function type that implements filters on the TrafficSplit lister
type TrafficSplitListOption func(o *TrafficSplitListOpt)

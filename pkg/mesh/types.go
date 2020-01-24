package mesh

import (
	"github.com/deislabs/smc/pkg/endpoint"
	TrafficTarget "github.com/deislabs/smi-sdk-go/pkg/apis/access/v1alpha1"
	TrafficSpec "github.com/deislabs/smi-sdk-go/pkg/apis/specs/v1alpha1"
	TrafficSplit "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

// ClientIdentity is the identity of an Envoy proxy connected to the Service Mesh Controller.
type ClientIdentity string

// Topology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type Topology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*TrafficSplit.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []endpoint.ServiceName

	// GetService fetches a specific service declared in SMI.
	GetService(endpoint.ServiceName) (service *v1.Service, exists bool, err error)

	// ListHTTPTrafficSpecs lists TrafficSpec SMI resources.
	ListHTTPTrafficSpecs() []*TrafficSpec.HTTPRouteGroup

	// ListTrafficTargets lists TrafficTarget SMI resources.
	ListTrafficTargets() []*TrafficTarget.TrafficTarget
}

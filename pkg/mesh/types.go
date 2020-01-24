package mesh

import (
	"github.com/deislabs/smc/pkg/endpoint"
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

// ClientIdentity is the identity of an Envoy proxy connected to the Service Mesh Controller.
type ClientIdentity string

// Topology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
// DEPRECATED
type Topology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []endpoint.ServiceName

	// GetService fetches a specific service declared in SMI.
	GetService(endpoint.ServiceName) (service *v1.Service, exists bool, err error)
}

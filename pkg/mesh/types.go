package mesh

import (
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
)

type ClientIdentity string

// ServiceName is a type for a service name
type ServiceName string

func (sn ServiceName) String() string {
	return string(sn)
}

// WeightedService is a struct of a delegated service backing a target service
type WeightedService struct {
	ServiceName ServiceName `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	IPs         []IP        `json:"ips:omitempty"`
}

// IP is an IP address
type IP string

// Topology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type Topology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []ServiceName

	// GetComputeIDForService is deprecated
	// Deprecated: this needs to be removed
	GetComputeIDForService(ServiceName) []ComputeID
}

// AzureID is a string type alias, which is the URI of a unique Azure cloud resource.
type AzureID string

// KubernetesID is a struct type, which points to a uniquely identifiable Kubernetes cluster.
type KubernetesID struct {
	ClusterID string
	Namespace string
	Service   string
}

// ComputeID is a struct, which contains the unique IDs of hte compute clusters where certain service may have Endpoints in.
type ComputeID struct {
	AzureID
	KubernetesID
}

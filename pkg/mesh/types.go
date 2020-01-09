package mesh

import (
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

// IP is an IP address
type IP string

// Port is a numerical port of an Envoy proxy
type Port uint32

// ServiceName is a type for a service name
type ServiceName string

// ClientIdentity is the identity of an Envoy proxy connected to the Service Mesh Controller.
type ClientIdentity string

// Endpoint is a tuple of IP and Port, representing an Envoy proxy, fronting an instance of a service
type Endpoint struct {
	IP   `json:"ip"`
	Port `json:"port"`
}

// WeightedService is a struct of a delegated service backing a target service
type WeightedService struct {
	ServiceName ServiceName `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	Endpoints   []Endpoint  `json:"endpoints:omitempty"`
}

// Topology is an interface declaring functions, which provide the topology of a service mesh declared with SMI.
type Topology interface {
	// ListTrafficSplits lists TrafficSplit SMI resources.
	ListTrafficSplits() []*v1alpha2.TrafficSplit

	// ListServices fetches all services declared with SMI Spec.
	ListServices() []ServiceName

	// GetService fetches a specific service declared in SMI.
	GetService(ServiceName) (service *v1.Service, exists bool, err error)
}

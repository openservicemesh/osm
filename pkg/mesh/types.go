package mesh

import (
	"github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"
)

// ServiceCatalogI is an interface w/ requirements to implement a service catalog
type ServiceCatalogI interface {
	GetServiceIPs(svcName ServiceName) ([]IP, error)
	GetWeightedService(svcName ServiceName) ([]WeightedService, error)
	GetWeightedServices() (map[ServiceName][]WeightedService, error)
}

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

// ComputeProviderI interface to be implemented by Kubernetes, Azure etc. providers.
type ComputeProviderI interface {
	// Retrieve the IP addresses comprising the ServiceName.
	GetIPs(ServiceName) []IP
	GetID() string
	Run(<-chan struct{}) error
}

// SpecI is an interface declaring what an SMI spec provider should implement.
type SpecI interface {
	ListTrafficSplits() []*v1alpha2.TrafficSplit
	ListServices() []ServiceName
	GetComputeIDForService(ServiceName) []ComputeID
}

type AzureID string
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

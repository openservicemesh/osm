package mesh

import "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha2"

// ServiceCatalogI is an interface w/ requirements to implement a service catalog
type ServiceCatalogI interface {
	GetServiceIPs(svcName ServiceName) ([]IP, error)
	GetWeightedService(svcName ServiceName) ([]WeightedService, error)
	GetWeightedServices() (map[ServiceName][]WeightedService, error)
}

// ServiceName is a type for a service name
type ServiceName string

// ServiceProviderI is an interface declaring the required functions for any compute provider
type ServiceProviderI interface {
	GetIPs(svcName ServiceName) []IP
}

// WeightedService is a struct of a delegated service backing a target service
type WeightedService struct {
	ServiceName ServiceName `json:"service_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
	IPs         []IP        `json:"ips:omitempty"`
}

// IP is an IP address
type IP string

// ComputeProviderI is an interface declaring what a compute provider should implement.
type ComputeProviderI interface {
	GetIPs(svc ServiceName) []IP
	Run(stopCh <-chan struct{}) error
}

// SpecI is an interface declaring what an SMI spec provider should implement.
type SpecI interface {
	ListTrafficSplits() []*v1alpha2.TrafficSplit
	ListServices() []ServiceName
}

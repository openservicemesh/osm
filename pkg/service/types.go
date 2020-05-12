package service

import (
	"fmt"
	"strings"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

// Name is a type for a service name
type Name string

func (s Name) String() string {
	return string(s)
}

// NamespacedService is a type for a namespaced service
type NamespacedService struct {
	Namespace string
	Service   string
}

func (ns NamespacedService) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.Service)
}

// Account is a type for a service account
type Account string

func (s Account) String() string {
	return string(s)
}

// GetCommonName returns the Subject CN for the NamespacedService to be used for its certificate.
func (ns NamespacedService) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ns.Service, ns.Namespace, "svc", "cluster", "local"}, "."))
}

// NamespacedServiceAccount is a type for a namespaced service account
type NamespacedServiceAccount struct {
	Namespace      string
	ServiceAccount string
}

func (ns NamespacedServiceAccount) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.ServiceAccount)
}

// ClusterName is a type for a service name
type ClusterName string

//WeightedService is a struct of a service name, its weight and domain
type WeightedService struct {
	ServiceName NamespacedService `json:"service_name:omitempty"`
	Weight      int               `json:"weight:omitempty"`
	Domain      string            `json:"domain:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}

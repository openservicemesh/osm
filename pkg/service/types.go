package service

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// separator used upon marshalling/unmarshalling Namespaced Service to a string
	// or viceversa
	separator = "/"
)

// NamespacedService is a type for a namespaced service
type NamespacedService struct {
	Namespace string
	Service   string
}

func (ns NamespacedService) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.Service)
}

//Equals checks if two namespaced services are equal
func (ns NamespacedService) Equals(service NamespacedService) bool {
	return reflect.DeepEqual(ns, service)
}

// UnmarshalNamespacedService unmarshals a NamespaceService type from a string
func UnmarshalNamespacedService(str string) (*NamespacedService, error) {
	slices := strings.Split(str, separator)
	if len(slices) != 2 {
		return nil, errInvalidNamespacedServiceFormat
	}

	// Make sure the slices are not empty. Split might actually leave empty slices.
	for _, sep := range slices {
		if len(sep) == 0 {
			return nil, errInvalidNamespacedServiceFormat
		}
	}

	return &NamespacedService{
		Namespace: slices[0],
		Service:   slices[1],
	}, nil
}

// GetCommonName returns the Subject CN for the NamespacedService to be used for its certificate.
func (ns NamespacedService) GetCommonName() certificate.CommonName {
	return certificate.CommonName(strings.Join([]string{ns.Service, ns.Namespace, "svc", "cluster", "local"}, "."))
}

// K8sServiceAccount is a type for a namespaced service account
type K8sServiceAccount struct {
	Namespace string
	Name      string
}

func (ns K8sServiceAccount) String() string {
	return fmt.Sprintf("%s/%s", ns.Namespace, ns.Name)
}

// ClusterName is a type for a service name
type ClusterName string

//WeightedService is a struct of a service name, its weight and domain
type WeightedService struct {
	NamespacedService NamespacedService `json:"service_name:omitempty"`
	Weight            int               `json:"weight:omitempty"`
	Domain            string            `json:"domain:omitempty"`
}

// WeightedCluster is a struct of a cluster and is weight that is backing a service
type WeightedCluster struct {
	ClusterName ClusterName `json:"cluster_name:omitempty"`
	Weight      int         `json:"weight:omitempty"`
}

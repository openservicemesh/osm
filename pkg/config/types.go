// Package config implements the Kubernetes client for the resources in the multiclusterservice.openservicemesh.io API group
package config

import (
	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	configv1alpha3Client "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("multiclusterservice-controller")
)

// client is the type used to represent the Kubernetes client for the multiclusterservice.openservicemesh.io API group
type client struct {
	informer       configv1alpha3Client.MultiClusterServiceInformer
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the multiclusterservice.openservicemesh.io API group
type Controller interface {
	ListMultiClusterServices() []configv1alpha3.MultiClusterService
	GetMultiClusterService(name, namespace string) *configv1alpha3.MultiClusterService
	GetMultiClusterServiceByServiceAccount(serviceAccount, namespace string) []configv1alpha3.MultiClusterService
}

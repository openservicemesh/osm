// Package config implements the Kubernetes client for the resources in the multiclusterservice.openservicemesh.io API group
package config

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configV1alpha1Informers "github.com/openservicemesh/osm/pkg/gen/client/config/informers/externalversions/config/v1alpha1"
	k8sInterfaces "github.com/openservicemesh/osm/pkg/k8s/interfaces"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("multiclusterservice-controller")
)

// client is the type used to represent the Kubernetes client for the multiclusterservice.openservicemesh.io API group
type client struct {
	informer       configV1alpha1Informers.MultiClusterServiceInformer
	kubeController k8sInterfaces.Controller
}

// Controller is the interface for the functionality provided by the resources part of the multiclusterservice.openservicemesh.io API group
type Controller interface {
	ListMultiClusterServices() []v1alpha1.MultiClusterService
	GetMultiClusterService(name, namespace string) *v1alpha1.MultiClusterService
	GetMultiClusterServiceByServiceAccount(serviceAccount, namespace string) []v1alpha1.MultiClusterService
}

// Package config implements the Kubernetes client for the resources in the multiclusterservice.openservicemesh.io API group
package config

import (
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	kubernetes "github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("multiclusterservice-controller")
)

// informerCollection is the type used to represent the collection of informers for the multiclusterservice.openservicemesh.io API group
type informerCollection struct {
	multiClusterService cache.SharedIndexInformer
}

// cacheCollection is the type used to represent the collection of caches for the multiclusterservice.openservicemesh.io API group
type cacheCollection struct {
	multiClusterService cache.Store
}

// client is the type used to represent the Kubernetes client for the multiclusterservice.openservicemesh.io API group
type client struct {
	informers      *informerCollection
	caches         *cacheCollection
	kubeController kubernetes.Controller
}

// Controller is the interface for the functionality provided by the resources part of the multiclusterservice.openservicemesh.io API group
type Controller interface {
	// TODO: specify required functions
	ListMultiClusterServices() []*v1alpha1.MultiClusterService
	GetMultiClusterService(name, namespace string) []*v1alpha1.MultiClusterService
}

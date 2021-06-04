// Package config implements the Kubernetes client for the resources in the remoteservice.openservicemesh.io API group
package config

import (
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	
	log = logger.New("multiclusterservice-controller")
)

// informerCollection is the type used to represent the collection of informers for the remoteservice.openservicemesh.io API group
type informerCollection struct {
	remoteService cache.SharedIndexInformer
}

// cacheCollection is the type used to represent the collection of caches for the remoteservice.openservicemesh.io API group
type cacheCollection struct {
	remoteService cache.Store
}

// client is the type used to represent the Kubernetes client for the remoteservice.openservicemesh.io API group
type client struct {
	informers      *informerCollection
	caches         *cacheCollection
	cacheSynced    chan interface{}
	kubeController kubernetes.Controller
}

// Controller is the interface for the functionality provided by the resources part of the remoteservice.openservicemesh.io API group
type Controller interface {
	// TODO: specify required functions
}

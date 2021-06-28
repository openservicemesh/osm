// Package policy implements the Kubernetes client for the resources in the policy.openservicemesh.io API group
package policy

import (
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
)

var (
	log = logger.New("policy-controller")
)

// informerCollection is the type used to represent the collection of informers for the policy.openservicemesh.io API group
type informerCollection struct {
	egress cache.SharedIndexInformer
}

// cacheCollection is the type used to represent the collection of caches for the policy.openservicemesh.io API group
type cacheCollection struct {
	egress cache.Store
}

// client is the type used to represent the Kubernetes client for the policy.openservicemesh.io API group
type client struct {
	informers      *informerCollection
	caches         *cacheCollection
	cacheSynced    chan interface{}
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the policy.openservicemesh.io API group
type Controller interface {
	// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity
	ListEgressPoliciesForSourceIdentity(identity.K8sServiceAccount) []*policyV1alpha1.Egress
}

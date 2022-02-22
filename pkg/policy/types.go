// Package policy implements the Kubernetes client for the resources in the policy.openservicemesh.io API group
package policy

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	policyV1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("policy-controller")
)

// informerCollection is the type used to represent the collection of informers for the policy.openservicemesh.io API group
type informerCollection struct {
	egress                 cache.SharedIndexInformer
	ingressBackend         cache.SharedIndexInformer
	retry                  cache.SharedIndexInformer
	upstreamTrafficSetting cache.SharedIndexInformer
}

// cacheCollection is the type used to represent the collection of caches for the policy.openservicemesh.io API group
type cacheCollection struct {
	egress                 cache.Store
	ingressBackend         cache.Store
	retry                  cache.Store
	upstreamTrafficSetting cache.Store
}

// client is the type used to represent the Kubernetes client for the policy.openservicemesh.io API group
type client struct {
	informers      *informerCollection
	caches         *cacheCollection
	kubeController k8s.Controller
}

// Controller is the interface for the functionality provided by the resources part of the policy.openservicemesh.io API group
type Controller interface {
	// ListEgressPoliciesForSourceIdentity lists the Egress policies for the given source identity
	ListEgressPoliciesForSourceIdentity(identity.K8sServiceAccount) []*policyV1alpha1.Egress

	// GetIngressBackendPolicy returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicy(service.MeshService) *policyV1alpha1.IngressBackend

	// ListRetryPolicies returns the Retry policies for the given source identity
	ListRetryPolicies(identity.K8sServiceAccount) []*policyV1alpha1.Retry

	// GetUpstreamTrafficSetting returns the UpstreamTrafficSetting resource that matches the given options
	GetUpstreamTrafficSetting(UpstreamTrafficSettingGetOpt) *policyv1alpha1.UpstreamTrafficSetting
}

// UpstreamTrafficSettingGetOpt specifies the options used to filter UpstreamTrafficSetting objects as a part of its getter
type UpstreamTrafficSettingGetOpt struct {
	// MeshService specifies the mesh service known within the cluster
	// Must be specified when retrieving a resource matching the upstream
	// mesh service.
	MeshService *service.MeshService

	// NamespacedName specifies the name and namespace of the resource
	NamespacedName *types.NamespacedName
}

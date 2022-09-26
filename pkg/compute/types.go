package compute

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
)

// Interface is an interface to be implemented by components abstracting Kubernetes, and other compute/cluster providers
type Interface interface {
	k8s.PassthroughInterface
	// GetSecret returns the secret for a given namespace and secret name
	GetSecret(string, string) *models.Secret

	// ListSecrets returns a list of secrets
	ListSecrets() []*models.Secret

	// UpdateSecret updates the given secret
	UpdateSecret(context.Context, *models.Secret) error

	// GetMeshService returns the service.MeshService corresponding to the Port used by clients
	// to communicate with it
	GetMeshService(name, namespace string, port uint16) (service.MeshService, error)

	// GetServicesForServiceIdentity retrieves the namespaced services for a given service identity
	GetServicesForServiceIdentity(identity.ServiceIdentity) []service.MeshService

	// ListServices returns a list of services that are part of monitored namespaces
	ListServices() []service.MeshService

	// ListServiceIdentitiesForService returns service identities for given service
	ListServiceIdentitiesForService(name, namespace string) ([]identity.ServiceIdentity, error)

	// ListEndpointsForService retrieves the IP addresses comprising the given service.
	ListEndpointsForService(service.MeshService) []endpoint.Endpoint

	// ListEndpointsForIdentity retrieves the list of IP addresses for the given service account
	ListEndpointsForIdentity(identity.ServiceIdentity) []endpoint.Endpoint

	// GetResolvableEndpointsForService returns the expected endpoints that are to be reached when the service FQDN is resolved under
	// the scope of the provider
	GetResolvableEndpointsForService(service.MeshService) []endpoint.Endpoint

	IsMetricsEnabled(*envoy.Proxy) (bool, error)

	GetHostnamesForService(svc service.MeshService, localNamespace bool) []string

	// ListServicesForProxy gets the services that map to the given proxy.
	ListServicesForProxy(p *envoy.Proxy) ([]service.MeshService, error)

	// ListEgressPoliciesForServiceAccount lists the Egress policies for the given source identity based on service accounts
	ListEgressPoliciesForServiceAccount(sa identity.K8sServiceAccount) []*policyv1alpha1.Egress

	// GetIngressBackendPolicyForService returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicyForService(svc service.MeshService) *policyv1alpha1.IngressBackend

	// ListRetryPoliciesForServiceAccount returns the retry policies for the given source identity based on service accounts.
	ListRetryPoliciesForServiceAccount(source identity.K8sServiceAccount) []*policyv1alpha1.Retry

	// GetUpstreamTrafficSettingByNamespace returns the UpstreamTrafficSetting resource that matches the namespace
	GetUpstreamTrafficSettingByNamespace(ns *types.NamespacedName) *policyv1alpha1.UpstreamTrafficSetting

	// GetUpstreamTrafficSettingByService returns the UpstreamTrafficSetting resource that matches the given service
	GetUpstreamTrafficSettingByService(meshService *service.MeshService) *policyv1alpha1.UpstreamTrafficSetting

	// GetUpstreamTrafficSettingByHost returns the UpstreamTrafficSetting resource that matches the host
	GetUpstreamTrafficSettingByHost(host string) *policyv1alpha1.UpstreamTrafficSetting

	GetProxyStatsHeaders(p *envoy.Proxy) (map[string]string, error)

	// VerifyProxy attempts to lookup a pod that matches the given proxy instance by service identity, namespace, and UUID
	VerifyProxy(proxy *envoy.Proxy) error

	// ListNamespaces returns the namespaces monitored by the mesh
	ListNamespaces() ([]string, error)
}

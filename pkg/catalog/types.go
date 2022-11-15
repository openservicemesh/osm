// Package catalog implements the MeshCatalog, which forms the central component in OSM that transforms
// outputs from all other components (SMI policies, Kubernetes services, endpoints etc.) into configuration that is
// consumed by the the proxy control plane component to program sidecar proxies.
// Reference: https://github.com/openservicemesh/osm/blob/main/DESIGN.md#5-mesh-catalog
package catalog

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/service"
)

var (
	log = logger.New("mesh-catalog")
)

type trafficDirection string

const (
	inbound  trafficDirection = "inbound"
	outbound trafficDirection = "outbound"
)

// Interface is an interface to be implemented by components abstracting Kubernetes, and other compute/cluster providers
type Interface interface {
	k8s.PassthroughInterface

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

	IsMetricsEnabled(*models.Proxy) (bool, error)

	GetHostnamesForService(svc service.MeshService, localNamespace bool) []string

	// ListServicesForProxy gets the services that map to the given proxy.
	ListServicesForProxy(p *models.Proxy) ([]service.MeshService, error)

	// ListEgressPoliciesForServiceAccount lists the Egress policies for the given source identity based on service accounts
	ListEgressPoliciesForServiceAccount(sa identity.K8sServiceAccount) []*v1alpha1.Egress

	// GetIngressBackendPolicyForService returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicyForService(svc service.MeshService) *v1alpha1.IngressBackend

	// ListRetryPoliciesForServiceAccount returns the retry policies for the given source identity based on service accounts.
	ListRetryPoliciesForServiceAccount(source identity.K8sServiceAccount) []*v1alpha1.Retry

	// GetUpstreamTrafficSettingByNamespace returns the UpstreamTrafficSetting resource that matches the namespace
	GetUpstreamTrafficSettingByNamespace(ns *types.NamespacedName) *v1alpha1.UpstreamTrafficSetting

	// GetUpstreamTrafficSettingByService returns the UpstreamTrafficSetting resource that matches the given service
	GetUpstreamTrafficSettingByService(meshService *service.MeshService) *v1alpha1.UpstreamTrafficSetting

	// GetUpstreamTrafficSettingByHost returns the UpstreamTrafficSetting resource that matches the host
	GetUpstreamTrafficSettingByHost(host string) *v1alpha1.UpstreamTrafficSetting

	GetProxyStatsHeaders(p *models.Proxy) (map[string]string, error)

	// GetProxyConfig takes the given proxy, port forwards to the pod from this proxy, and returns the envoy config
	GetProxyConfig(proxy *models.Proxy, configType string, kubeConfig *rest.Config) (string, error)

	// VerifyProxy attempts to lookup a pod that matches the given proxy instance by service identity, namespace, and UUID
	VerifyProxy(proxy *models.Proxy) error

	// ListNamespaces returns the namespaces monitored by the mesh
	ListNamespaces() ([]string, error)

	// GetTelemetryConfig returns the Telemetry config for the given proxy instance.
	// It returns the most specific match if multiple matching policies exist, in the following
	// order of preference: 1. selector match, 2. namespace match, 3. global match
	GetTelemetryConfig(*models.Proxy) models.TelemetryConfig

	// GetMeshConfig returns the current MeshConfig
	GetMeshConfig() v1alpha2.MeshConfig
}

package validator

import (
	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/logger"
	"github.com/openservicemesh/osm/pkg/service"
)

var log = logger.New(ValidatorWebhookSvc)

type ValidatorInfraClient interface {
	// GetUpstreamTrafficSettingByHost returns the UpstreamTrafficSetting resource that matches the host
	GetUpstreamTrafficSettingByHost(host string) *v1alpha1.UpstreamTrafficSetting

	// GetIngressBackendPolicyForService returns the IngressBackend policy for the given backend MeshService
	GetIngressBackendPolicyForService(svc service.MeshService) *v1alpha1.IngressBackend

	ListMeshRootCertificates() ([]*configv1alpha2.MeshRootCertificate, error)
}

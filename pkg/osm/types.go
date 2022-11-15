package osm

import (
	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/models"
)

type ControlPlaneInfraClient interface {
	// GetMeshConfig returns the current MeshConfig
	GetMeshConfig() v1alpha2.MeshConfig

	// VerifyProxy attempts to lookup a pod that matches the given proxy instance by service identity, namespace, and UUID
	VerifyProxy(proxy *models.Proxy) error
}

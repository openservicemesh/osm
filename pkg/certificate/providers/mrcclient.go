package providers

import (
	"fmt"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/messaging"
)

type MRCClientImpl struct {
	configurator configurator.Configurator
	msgBroker    *messaging.Broker
	MRCProviderGenerator
}

func (m *MRCClientImpl) List() ([]v1alpha2.MeshRootCertificate, error) {
	return m.configurator.GetMeshRootCertificates()
}

func (m *MRCClientImpl) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, string, error) {
	p := mrc.Spec.Provider
	switch {
	case p.Tresor != nil:
		return m.getTresorOSMCertificateManager(mrc)
	case p.Vault != nil:
		return m.getHashiVaultOSMCertificateManager(mrc)
	case p.CertManager != nil:
		return m.getCertManagerOSMCertificateManager(mrc)
	default:
		return nil, "", fmt.Errorf("unknown certificate provider: %+v", p)
	}
}

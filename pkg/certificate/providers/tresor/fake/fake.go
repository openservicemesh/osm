package fake

import (
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	rootCertOrganization = "Open Service Mesh Tresor"
)

type fakeMRCClient struct{}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, string, error) {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	ca, err := tresor.NewCA("Fake Tresor CN", 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		return nil, "", err
	}
	issuer, err := tresor.New(ca, rootCertOrganization, 2048)
	return issuer, "issuer-1", err
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha2.MeshRootCertificate{{}}, nil
}

// NewFake constructs a fake certificate client using a certificate
func NewFake(msgBroker *messaging.Broker) *certificate.Manager {
	validityPeriod := func() time.Duration { return 1 * time.Hour }
	tresorCertManager, err := certificate.NewManager(&fakeMRCClient{}, validityPeriod, msgBroker)
	if err != nil {
		return nil
	}
	return tresorCertManager
}

// NewFakeCertificate is a helper creating Certificates for unit tests.
func NewFakeCertificate() *certificate.Certificate {
	return &certificate.Certificate{
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		IssuingCA:    pem.RootCertificate("xx"),
		Expiration:   time.Now(),
		CommonName:   "foo.bar.co.uk",
		SerialNumber: "-the-certificate-serial-number-",
	}
}

package certificate

import (
	"fmt"
	time "time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var (
	validity = time.Hour
)

type fakeMRCClient struct{}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, string, error) {
	return &fakeIssuer{}, "fake-issuer-1", nil
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha2.MeshRootCertificate{{}}, nil
}

type fakeIssuer struct {
	err bool
	id  string
}

// IssueCertificate is a testing helper to satisfy the certificate client interface
func (i *fakeIssuer) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	if i.err {
		return nil, fmt.Errorf("%s failed", i.id)
	}
	return &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(validityPeriod),
		// simply used to distinguish the private/public key from other issuers
		IssuingCA:  pem.RootCertificate(i.id),
		PrivateKey: pem.PrivateKey(i.id),
	}, nil
}

// FakeCertManager is a testing helper that returns a *certificate.Manager
func FakeCertManager() (*Manager, error) {
	cm, err := NewManager(
		&fakeMRCClient{},
		func() time.Duration { return validity },
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}
	return cm, nil
}

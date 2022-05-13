package certificate

import (
	"fmt"
	time "time"

	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

var (
	caCert = &Certificate{
		CommonName: "Test CA",
		Expiration: time.Now().Add(time.Hour * 24),
	}
	validity = time.Hour
)

type fakeMRCClient struct{}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, error) {
	return &fakeIssuer{}, nil
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha2.MeshRootCertificate{{}}, nil
}

// AddEventHandler is a no-op for the legacy client. The previous client could not handle changes, but we need this
// method to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) AddEventHandler(cache.ResourceEventHandler) {}

type fakeIssuer struct{}

// IssueCertificate is a testing helper to satisfy the certificate client interface
func (i *fakeIssuer) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	return &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(validityPeriod),
	}, nil
}

// FakeCertManager is a testing helper that returns a *certificate.Manager
func FakeCertManager() (*Manager, error) {
	cm, err := NewManager(
		&fakeMRCClient{},
		validity,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}
	return cm, nil
}

package certificate

import (
	"context"
	"fmt"
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var (
	validity = time.Hour
)

// fakeMRCClient implements the MRCClient interface
type fakeMRCClient struct {
	mrcList []*v1alpha2.MeshRootCertificate
}

// GetCertIssuerForMRC returns a fakeIssuer and pre-generated RootCertificate. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error) {
	return &fakeIssuer{id: mrc.Name}, pem.RootCertificate("rootCA"), nil
}

// ListMeshRootCertificates returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return c.mrcList, nil
}

// UpdateMeshRootCertificate updates the given mesh root certificate.
func (c *fakeMRCClient) UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) error {
	// TODO(5046): implement this.
	return nil
}

// Watch returns a channel that has one MRCEventAdded. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan MRCEvent, error) {
	ch := make(chan MRCEvent)
	go func() {
		ch <- MRCEvent{
			MRCName: "osm-mesh-root-certificate",
		}
		close(ch)
	}()

	return ch, nil
}

type fakeIssuer struct {
	err bool
	id  string
}

// IssueCertificate is a testing helper to satisfy the certificate client interface
func (i *fakeIssuer) IssueCertificate(options IssueOptions) (*Certificate, error) {
	if i.err {
		return nil, fmt.Errorf("%s failed", i.id)
	}
	return &Certificate{
		CommonName: options.CommonName(),
		Expiration: time.Now().Add(options.ValidityDuration),
		// simply used to distinguish the private/public key from other issuers
		IssuingCA:  pem.RootCertificate(i.id),
		TrustedCAs: pem.RootCertificate(i.id),
		PrivateKey: pem.PrivateKey(i.id),
	}, nil
}

// FakeCertManager is a testing helper that returns a *certificate.Manager
func FakeCertManager() (*Manager, error) {
	getCertValidityDuration := func() time.Duration { return validity }
	cm, err := NewManager(
		context.Background(),
		&fakeMRCClient{},
		getCertValidityDuration,
		getCertValidityDuration,
		1*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}

	return cm, nil
}

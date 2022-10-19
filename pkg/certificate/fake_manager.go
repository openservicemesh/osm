package certificate

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	osmConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
)

// fakeMRCClient implements the MRCClient interface
type fakeMRCClient struct {
	configClient osmConfigClient.Interface
}

// GetCertIssuerForMRC returns a fakeIssuer and pre-generated RootCertificate. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error) {
	return &fakeIssuer{id: mrc.Name}, pem.RootCertificate("rootCA"), nil
}

// ListMeshRootCertificates returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	mrcList, err := c.configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var mrcListPointers []*v1alpha2.MeshRootCertificate
	for i := range mrcList.Items {
		mrcListPointers = append(mrcListPointers, &mrcList.Items[i])
	}
	return mrcListPointers, nil
}

// UpdateMeshRootCertificate updates the given mesh root certificate.
func (c *fakeMRCClient) UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) error {
	_, err := c.configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Update(context.Background(), mrc, v1.UpdateOptions{})
	return err
}

// Watch returns a channel that has one MRCEventAdded. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan MRCEvent, error) {
	// send event for first CA created
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

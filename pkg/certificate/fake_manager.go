package certificate

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	validity = time.Hour
)

// fakeMRCClient implements the MRCClient interface
type fakeMRCClient struct {
	compute.Interface
}

// GetCertIssuerForMRC returns a fakeIssuer and pre-generated RootCertificate. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error) {
	return &fakeIssuer{}, pem.RootCertificate("rootCA"), nil
}

// ListMeshRootCertificates returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha2.MeshRootCertificate{{
		Spec: v1alpha2.MeshRootCertificateSpec{
			TrustDomain: "fake.domain.com",
		},
	}}, nil
}

// UpdateMeshRootCertificate updates the given mesh root certificate.
func (c *fakeMRCClient) UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	// TODO(5046): implement this.
	return nil, nil
}

// GetMeshRootCertificate gets the specified mesh root certificate.
func (c *fakeMRCClient) GetMeshRootCertificate(mrcName string) *v1alpha2.MeshRootCertificate {
	// TODO(5046): implement this.
	return nil
}

// Watch returns a channel that has one MRCEventAdded. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan MRCEvent, error) {
	ch := make(chan MRCEvent)
	go func() {
		ch <- MRCEvent{
			Type: MRCEventAdded,
			MRC: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					TrustDomain: "fake.domain.com",
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   v1alpha2.Ready,
							Status: v1.ConditionUnknown,
						},
						{
							Type:   v1alpha2.Accepted,
							Status: v1.ConditionUnknown,
						},
						{
							Type:   v1alpha2.IssuingRollout,
							Status: v1.ConditionUnknown,
						},
						{
							Type:   v1alpha2.ValidatingRollout,
							Status: v1.ConditionUnknown,
						},
						{
							Type:   v1alpha2.IssuingRollback,
							Status: v1.ConditionUnknown,
						},
						{
							Type:   v1alpha2.ValidatingRollback,
							Status: v1.ConditionUnknown,
						},
					},
				},
			},
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
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}

	return cm, nil
}

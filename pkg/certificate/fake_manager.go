package certificate

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	validity = time.Hour
)

type fakeMRCClient struct {
	fakeConfigClient *fakeConfigClientset.Clientset
}

func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (Issuer, pem.RootCertificate, error) {
	return &fakeIssuer{}, pem.RootCertificate("rootCA"), nil
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) List() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	return []*v1alpha2.MeshRootCertificate{{
		Spec: v1alpha2.MeshRootCertificateSpec{
			TrustDomain: "fake.domain.com",
		},
	}}, nil
}

func (c *fakeMRCClient) Get(name string) *v1alpha2.MeshRootCertificate {
	mrc, _ := c.fakeConfigClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Get(context.Background(), name, metav1.GetOptions{})
	return mrc
}

func (c *fakeMRCClient) Update(obj *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	return c.fakeConfigClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Update(context.Background(), obj, metav1.UpdateOptions{})
}

func (c *fakeMRCClient) UpdateStatus(obj *v1alpha2.MeshRootCertificate) (*v1alpha2.MeshRootCertificate, error) {
	return c.fakeConfigClient.ConfigV1alpha2().MeshRootCertificates("osm-system").UpdateStatus(context.Background(), obj, metav1.UpdateOptions{})
}

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
					ComponentStatuses: v1alpha2.MeshRootCertificateComponentStatuses{
						Webhooks:        constants.MRCComponentStatusUnknown,
						XDSControlPlane: constants.MRCComponentStatusUnknown,
						Sidecar:         constants.MRCComponentStatusUnknown,
						Bootstrap:       constants.MRCComponentStatusUnknown,
						Gateway:         constants.MRCComponentStatusUnknown,
					},
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeReady,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollback,
							Status: constants.MRCConditionStatusUnknown,
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
	)
	if err != nil {
		return nil, fmt.Errorf("error creating fakeCertManager, err: %w", err)
	}
	return cm, nil
}

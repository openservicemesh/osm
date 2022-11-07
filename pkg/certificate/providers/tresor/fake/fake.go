// Package fake moves fakes to their own sub-package
package fake

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	osmConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

const (
	rootCertOrganization = "Open Service Mesh Tresor"
	initialRootName      = constants.DefaultMeshRootCertificateName
	defaultNamespace     = "osm-system"
)

var defaultMRC *v1alpha2.MeshRootCertificate = &v1alpha2.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Name:      initialRootName,
		Namespace: defaultNamespace,
	},
	Spec: v1alpha2.MeshRootCertificateSpec{
		TrustDomain: "cluster.local",
		Intent:      v1alpha2.ActiveIntent,
	},
}

type fakeMRCClient struct {
	mrcChannel   chan certificate.MRCEvent
	configClient osmConfigClient.Interface
}

// NewFakeMRC allows for publishing events on to the watch channel to generate MRC events
// and a default MRC is created
func NewFakeMRC() *fakeMRCClient { //nolint: revive // unexported-return
	ch := make(chan certificate.MRCEvent)
	configClient := configFake.NewSimpleClientset([]runtime.Object{defaultMRC}...)

	return &fakeMRCClient{
		mrcChannel:   ch,
		configClient: configClient,
	}
}

// NewFakeMRCWithConfig allows for publishing events on to the watch channel to generate MRC events
func NewFakeMRCWithConfig(configClient osmConfigClient.Interface) *fakeMRCClient { //nolint: revive // unexported-return
	ch := make(chan certificate.MRCEvent)

	return &fakeMRCClient{
		mrcChannel:   ch,
		configClient: configClient,
	}
}

// NewCertEvent allows pushing MRC events which can trigger cert changes
func (c *fakeMRCClient) NewCertEvent(mrcName string) {
	c.mrcChannel <- certificate.MRCEvent{
		MRCName: mrcName,
	}
}

// UpdateMeshRootCertificate is not implemented on the compat client and always returns an error
func (c *fakeMRCClient) UpdateMeshRootCertificate(mrc *v1alpha2.MeshRootCertificate) error {
	_, err := c.configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Update(context.Background(), mrc, metav1.UpdateOptions{})
	return err
}

// GetCertIssuerForMRC will return a root cert for testing.
func (c *fakeMRCClient) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, pem.RootCertificate, error) {
	rootCertCountry := "US"
	rootCertLocality := "CA"
	cn := certificate.CommonName(mrc.Name)

	ca, err := tresor.NewCA(cn, 1*time.Hour, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		return nil, nil, err
	}
	issuer, err := tresor.New(ca, rootCertOrganization, 2048)
	if err != nil {
		return nil, nil, err
	}

	cert, err := issuer.IssueCertificate(certificate.NewCertOptionsWithFullName("rootCA", 24*time.Hour))
	if err != nil {
		return nil, nil, err
	}
	return issuer, cert.GetTrustedCAs(), nil
}

// List returns the single, pre-generated MRC. It is intended to implement the certificate.MRCClient interface.
func (c *fakeMRCClient) ListMeshRootCertificates() ([]*v1alpha2.MeshRootCertificate, error) {
	// return single empty object in the list.
	mrcList, err := c.configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var mrcListPointers []*v1alpha2.MeshRootCertificate
	for i := range mrcList.Items {
		mrcListPointers = append(mrcListPointers, &mrcList.Items[i])
	}
	return mrcListPointers, nil
}

func (c *fakeMRCClient) Watch(ctx context.Context) (<-chan certificate.MRCEvent, error) {
	// send event for first CA created
	go func() {
		c.NewCertEvent(initialRootName)
	}()

	return c.mrcChannel, nil
}

// NewFake constructs a fake certificate client using a certificate
func NewFake(checkInterval time.Duration) *certificate.Manager {
	getValidityDuration := func() time.Duration { return 1 * time.Hour }
	return NewFakeWithValidityDuration(getValidityDuration, checkInterval)
}

// NewFakeWithValidityDuration constructs a fake certificate manager with specified cert validity duration
func NewFakeWithValidityDuration(getCertValidityDuration func() time.Duration, checkInterval time.Duration) *certificate.Manager {
	tresorCertManager, err := certificate.NewManager(context.Background(), NewFakeMRC(), getCertValidityDuration, getCertValidityDuration, checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("error encountered creating fake cert manager")
		return nil
	}
	return tresorCertManager
}

// NewFakeWithMRCClient constructs a fake certificate manager with specified cert validity duration and fake MRC client
func NewFakeWithMRCClient(fakeMRCClient *fakeMRCClient, checkInterval time.Duration) *certificate.Manager {
	getValidityDuration := func() time.Duration { return 1 * time.Hour }
	tresorCertManager, err := certificate.NewManager(context.Background(), fakeMRCClient, getValidityDuration, getValidityDuration, checkInterval)
	if err != nil {
		log.Error().Err(err).Msg("error encountered creating fake cert manager")
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
		TrustedCAs:   pem.RootCertificate("xx"),
		Expiration:   time.Now(),
		CommonName:   "foo.bar.co.uk",
		SerialNumber: "-the-certificate-serial-number-",
	}
}

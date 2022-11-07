package certificate

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cskr/pubsub"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/identity"
)

func TestShouldRotate(t *testing.T) {
	manager := &Manager{}

	testCases := []struct {
		name             string
		cert             *Certificate
		managerKeyIssuer *issuer
		managerPubIssuer *issuer
		expectedRotation bool
	}{
		{
			name: "Expired certificate",
			cert: &Certificate{
				Expiration:         time.Now().Add(-1 * time.Hour),
				signingIssuerID:    "1",
				validatingIssuerID: "1",
			},
			managerKeyIssuer: &issuer{ID: "1"},
			managerPubIssuer: &issuer{ID: "1"},
			expectedRotation: true,
		},
		{
			name: "Mismatched certificate",
			cert: &Certificate{
				Expiration:         time.Now().Add(1 * time.Hour),
				signingIssuerID:    "1",
				validatingIssuerID: "2",
			},
			managerKeyIssuer: &issuer{ID: "2"},
			managerPubIssuer: &issuer{ID: "1"},
			expectedRotation: true,
		},
		{
			name: "Valid certificate",
			cert: &Certificate{
				Expiration:         time.Now().Add(constants.OSMCertificateValidityPeriod),
				signingIssuerID:    "1",
				validatingIssuerID: "1",
			},
			managerKeyIssuer: &issuer{ID: "1"},
			managerPubIssuer: &issuer{ID: "1"},
			expectedRotation: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			manager.signingIssuer = tc.managerKeyIssuer
			manager.validatingIssuer = tc.managerPubIssuer

			rotate := manager.ShouldRotate(tc.cert)
			assert.Equal(tc.expectedRotation, rotate)
		})
	}
}

func TestRotor(t *testing.T) {
	assert := tassert.New(t)
	require := trequire.New(t)

	cnPrefix := "foo"
	// negative time means this cert has already expired -- will be rotated asap
	getServiceCertValidityPeriod := func() time.Duration { return -1 * time.Hour }
	getIngressGatewayCertValidityPeriod := func() time.Duration { return -1 * time.Hour }

	stop := make(chan struct{})
	defer close(stop)
	configClient := configFake.NewSimpleClientset([]runtime.Object{activeMRC1}...)
	certManager, err := NewManager(context.Background(), &fakeMRCClient{configClient: configClient},
		getServiceCertValidityPeriod, getIngressGatewayCertValidityPeriod, 5*time.Second)
	require.NoError(err)

	certA, err := certManager.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
	require.NoError(err)
	certRotateChan, unsub := certManager.SubscribeRotations(cnPrefix)
	defer unsub()

	// Wait for two certificate rotations to be announced and terminate
	<-certRotateChan
	newCert, err := certManager.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
	assert.NoError(err)
	assert.NotEqual(certA.GetExpiration(), newCert.GetExpiration())
	assert.NotEqual(certA, newCert)
}

func TestReleaseCertificate(t *testing.T) {
	cn := "Test CN"
	cert := &Certificate{
		CommonName: CommonName(cn),
		Expiration: time.Now().Add(1 * time.Hour),
	}

	manager := &Manager{}
	manager.cache.Store(cn, cert)

	testCases := []struct {
		name     string
		cnPrefix string
	}{
		{
			name:     "release existing certificate",
			cnPrefix: cn,
		},
		{
			name:     "release non-existing certificate",
			cnPrefix: cn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			manager.ReleaseCertificate(tc.cnPrefix)
			cert := manager.getFromCache(tc.cnPrefix)

			assert.Nil(cert)
		})
	}
}

func TestListIssuedCertificate(t *testing.T) {
	assert := tassert.New(t)

	cn := CommonName("Test Cert")
	cert := &Certificate{
		CommonName: cn,
	}

	anotherCn := CommonName("Another Test Cert")
	anotherCert := &Certificate{
		CommonName: anotherCn,
	}

	expectedCertificates := []*Certificate{cert, anotherCert}

	manager := &Manager{}
	manager.cache.Store(cn, cert)
	manager.cache.Store(anotherCn, anotherCert)

	cs := manager.ListIssuedCertificates()
	assert.Len(cs, 2)

	for i, c := range cs {
		match := false
		for _, ec := range expectedCertificates {
			if c.GetCommonName() == ec.GetCommonName() {
				match = true
				assert.Equal(ec, c)
				break
			}
		}

		if !match {
			t.Fatalf("Certificate #%v %v does not exist", i, c.GetCommonName())
		}
	}
}

func TestIssueCertificate(t *testing.T) {
	assert := tassert.New(t)
	cnPrefix := "fake-cert-cn"
	getServiceValidityDuration := func() time.Duration { return time.Hour }

	stop := make(chan struct{})
	defer close(stop)

	t.Run("single key issuer", func(t *testing.T) {
		cm := &Manager{
			serviceCertValidityDuration: getServiceValidityDuration,
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}, CertificateAuthority: pem.RootCertificate("id1"), TrustDomain: "fake1.domain.com"},
			validatingIssuer: &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}, CertificateAuthority: pem.RootCertificate("id1"), TrustDomain: "fake2.domain.com"},
			pubsub:           pubsub.New(0),
		}
		// single signingIssuer, not cached
		cert1, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotNil(cert1)
		assert.Equal(cert1.signingIssuerID, "id1")
		assert.Equal(cert1.validatingIssuerID, "id1")
		assert.Equal(cert1.GetIssuingCA(), pem.RootCertificate("id1"))
		assert.Equal(CommonName("fake-cert-cn.fake1.domain.com"), cert1.GetCommonName())

		// single signingIssuer cached
		cert2, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.Equal(cert1, cert2)
		assert.Equal(CommonName("fake-cert-cn.fake1.domain.com"), cert1.GetCommonName())

		// single key issuer, old version cached
		// TODO: could use informer logic to test mrc updates instead of just manually making changes.
		cm.signingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}, CertificateAuthority: pem.RootCertificate("id2"), TrustDomain: "fake2.domain.com"}
		cm.validatingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}, CertificateAuthority: pem.RootCertificate("id2")}

		cert3, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.Equal(CommonName("fake-cert-cn.fake2.domain.com"), cert3.GetCommonName())
		assert.NoError(err)
		assert.NotNil(cert3)
		assert.Equal(cert3.signingIssuerID, "id2")
		assert.Equal(cert3.validatingIssuerID, "id2")
		assert.NotEqual(cert2, cert3)
		assert.Equal(cert3.GetIssuingCA(), pem.RootCertificate("id2"))
	})

	t.Run("2 issuers", func(t *testing.T) {
		cm := &Manager{
			serviceCertValidityDuration: getServiceValidityDuration,
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}, CertificateAuthority: pem.RootCertificate("id1"), TrustDomain: "fake1.domain.com"},
			validatingIssuer: &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}, CertificateAuthority: pem.RootCertificate("id2"), TrustDomain: "fake2.domain.com"},
			pubsub:           pubsub.New(0),
		}

		// Not cached
		cert1, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotNil(cert1)
		assert.Equal(cert1.signingIssuerID, "id1")
		assert.Equal(cert1.validatingIssuerID, "id2")
		assert.Equal(pem.RootCertificate("id1"), cert1.GetIssuingCA())
		assert.Equal(pem.RootCertificate("id1id2"), cert1.GetTrustedCAs())
		assert.Equal(CommonName("fake-cert-cn.fake1.domain.com"), cert1.GetCommonName())

		// cached
		cert2, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.Equal(cert1, cert2)
		assert.Equal(CommonName("fake-cert-cn.fake1.domain.com"), cert2.GetCommonName())

		// cached, but validatingIssuer is removed
		cm.validatingIssuer = cm.signingIssuer
		cert3, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotEqual(cert1, cert3)
		assert.Equal(cert3.signingIssuerID, "id1")
		assert.Equal(cert3.validatingIssuerID, "id1")
		assert.Equal(cert3.GetIssuingCA(), pem.RootCertificate("id1"))
		assert.Equal(CommonName("fake-cert-cn.fake1.domain.com"), cert1.GetCommonName())

		// cached, but signingIssuer is old
		cm.signingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}, CertificateAuthority: pem.RootCertificate("id2"), TrustDomain: "fake2.domain.com"}
		cert4, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotEqual(cert3, cert4)
		assert.Equal(cert4.signingIssuerID, "id2")
		assert.Equal(cert4.validatingIssuerID, "id1")
		assert.Equal(pem.RootCertificate("id2"), cert4.GetIssuingCA())
		assert.Equal(pem.RootCertificate("id2id1"), cert4.GetTrustedCAs())
		assert.Equal(CommonName("fake-cert-cn.fake2.domain.com"), cert4.GetCommonName())

		// cached, but validatingIssuer is old
		cm.validatingIssuer = &issuer{ID: "id3", Issuer: &fakeIssuer{id: "id3"}, CertificateAuthority: pem.RootCertificate("id3"), TrustDomain: "fake3.domain.com"}
		cert5, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotEqual(cert4, cert5)
		assert.Equal(cert5.signingIssuerID, "id2")
		assert.Equal(cert5.validatingIssuerID, "id3")
		assert.Equal(pem.RootCertificate("id2"), cert5.GetIssuingCA())
		assert.Equal(pem.RootCertificate("id2id3"), cert5.GetTrustedCAs())
		assert.Equal(CommonName("fake-cert-cn.fake2.domain.com"), cert5.GetCommonName())
	})

	t.Run("bad issuers", func(t *testing.T) {
		cm := &Manager{
			serviceCertValidityDuration: getServiceValidityDuration,
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1", err: true}, CertificateAuthority: pem.RootCertificate("id1")},
			validatingIssuer: &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2", err: true}, CertificateAuthority: pem.RootCertificate("id2")},
			pubsub:           pubsub.New(0),
		}

		// bad signingIssuer
		cert, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.Nil(cert)
		assert.EqualError(err, "id1 failed")

		// bad validatingIssuer (should still succeed)
		cm.signingIssuer = &issuer{ID: "id3", Issuer: &fakeIssuer{id: "id3"}, CertificateAuthority: pem.RootCertificate("id3")}
		cert, err = cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.Equal(cert.signingIssuerID, "id3")
		assert.Equal(cert.validatingIssuerID, "id2")
		assert.Equal(pem.RootCertificate("id3"), cert.GetIssuingCA())
		assert.Equal(pem.RootCertificate("id3id2"), cert.GetTrustedCAs())

		// insert a cached cert
		cm.validatingIssuer = cm.signingIssuer
		cert, err = cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.NoError(err)
		assert.NotNil(cert)

		// bad signing cert on an existing cached cert, because the signingIssuer is new
		cm.signingIssuer = &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1", err: true}, CertificateAuthority: pem.RootCertificate("id1")}
		cert, err = cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix)))
		assert.EqualError(err, "id1 failed")
		assert.Nil(cert)
	})
}

func TestSubscribeRotations(t *testing.T) {
	assert := tassert.New(t)
	cnPrefix1 := "fake-cert-cn1"
	cnPrefix2 := "fake-cert-cn2"

	cm := &Manager{
		serviceCertValidityDuration: func() time.Duration { return time.Hour },
		// The root certificate signing all newly issued certificates
		signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}, CertificateAuthority: pem.RootCertificate("id1"), TrustDomain: "fake1.domain.com"},
		validatingIssuer: &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}, CertificateAuthority: pem.RootCertificate("id1"), TrustDomain: "fake1.domain.com"},
		pubsub:           pubsub.New(0),
	}

	ch1, unsub1 := cm.SubscribeRotations(cnPrefix1)
	ch2, unsub2 := cm.SubscribeRotations(cnPrefix2)
	defer unsub1()
	defer unsub2()

	cert, err := cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix1)))
	assert.NoError(err)
	assert.Equal("fake-cert-cn1.fake1.domain.com", cert.GetCommonName().String())

	cert, err = cm.IssueCertificate(ForServiceIdentity(identity.ServiceIdentity(cnPrefix2)))
	assert.NoError(err)
	assert.Equal("fake-cert-cn2.fake1.domain.com", cert.GetCommonName().String())

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		msg1 := <-ch1
		assert.Equal("fake-cert-cn1.fake2.domain.com", msg1.(*Certificate).GetCommonName().String())
		wg.Done()
	}()

	go func() {
		msg2 := <-ch2
		assert.Equal("fake-cert-cn2.fake2.domain.com", msg2.(*Certificate).GetCommonName().String())
		wg.Done()
	}()

	// swap the issuer which will trigger a rotation
	cm.signingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}, CertificateAuthority: pem.RootCertificate("id2"), TrustDomain: "fake2.domain.com"}
	cm.checkAndRotate()

	wg.Wait()
}

func TestManager_GetTrustDomain(t *testing.T) {
	tests := []struct {
		name                 string
		signingIssuer        *issuer
		validatingIssuer     *issuer
		expectedTrustDomains TrustDomain
		expectedAreDifferent bool
	}{
		{
			name:                 "should return both trustdomains",
			signingIssuer:        &issuer{TrustDomain: "cluster.local"},
			validatingIssuer:     &issuer{TrustDomain: "old.local"},
			expectedTrustDomains: TrustDomain{Signing: "cluster.local", Validating: "old.local"},
			expectedAreDifferent: true,
		},
		{
			name:                 "should return both trustdomains when the same",
			signingIssuer:        &issuer{TrustDomain: "cluster.local"},
			validatingIssuer:     &issuer{TrustDomain: "cluster.local"},
			expectedTrustDomains: TrustDomain{Signing: "cluster.local", Validating: "cluster.local"},
			expectedAreDifferent: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			m := &Manager{
				signingIssuer:    tt.signingIssuer,
				validatingIssuer: tt.validatingIssuer,
			}
			got := m.GetTrustDomains()
			assert.Equal(tt.expectedTrustDomains.Signing, got.Signing)
			assert.Equal(tt.expectedTrustDomains.Validating, got.Validating)
			assert.Equal(tt.expectedAreDifferent, got.AreDifferent())
		})
	}
}

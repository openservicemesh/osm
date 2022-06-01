package certificate

import (
	"testing"
	time "time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestRotor(t *testing.T) {
	assert := tassert.New(t)

	cn := CommonName("foo")
	validityPeriod := -1 * time.Hour // negative time means this cert has already expired -- will be rotated asap

	stop := make(chan struct{})
	defer close(stop)
	msgBroker := messaging.NewBroker(stop)
	certManager, err := NewManager(&fakeMRCClient{}, validityPeriod, msgBroker)
	certManager.Start(5*time.Second, stop)
	assert.NoError(err)

	certA, err := certManager.IssueCertificate(cn, validityPeriod)
	assert.NoError(err)
	certRotateChan := msgBroker.GetCertPubSub().Sub(announcements.CertificateRotated.String())

	// Wait for two certificate rotations to be announced and terminate
	<-certRotateChan
	newCert, err := certManager.IssueCertificate(cn, validityPeriod)
	assert.NoError(err)
	assert.NotEqual(certA.GetExpiration(), newCert.GetExpiration())
	assert.NotEqual(certA, newCert)
}

func TestReleaseCertificate(t *testing.T) {
	cn := CommonName("Test CN")
	cert := &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(1 * time.Hour),
	}

	manager := &Manager{}
	manager.cache.Store(cn, cert)

	testCases := []struct {
		name       string
		commonName CommonName
	}{
		{
			name:       "release existing certificate",
			commonName: cn,
		},
		{
			name:       "release non-existing certificate",
			commonName: cn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			manager.ReleaseCertificate(tc.commonName)
			cert := manager.getFromCache(tc.commonName)

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
	cn := CommonName("fake-cert-cn")
	assert := tassert.New(t)

	t.Run("single key issuer", func(t *testing.T) {
		cm := &Manager{
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}},
			validatingIssuer: &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}},
		}
		// single keyIssuer, not cached
		cert1, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotNil(cert1)
		assert.Equal(cert1.keyIssuerID, "id1")
		assert.Equal(cert1.pubIssuerID, "id1")
		assert.Equal(cert1.GetIssuingCA(), pem.RootCertificate("id1"))

		// single keyIssuer cached
		cert2, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.Equal(cert1, cert2)

		// single key issuer, old version cached
		// TODO: could use informer logic to test mrc updates instead of just manually making changes.
		cm.signingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}}
		cm.validatingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}}

		cert3, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotNil(cert3)
		assert.Equal(cert3.keyIssuerID, "id2")
		assert.Equal(cert3.pubIssuerID, "id2")
		assert.NotEqual(cert2, cert3)
		assert.Equal(cert3.GetIssuingCA(), pem.RootCertificate("id2"))
	})

	t.Run("2 issuers", func(t *testing.T) {
		cm := &Manager{
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1"}},
			validatingIssuer: &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}},
		}

		// Not cached
		cert1, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotNil(cert1)
		assert.Equal(cert1.keyIssuerID, "id1")
		assert.Equal(cert1.pubIssuerID, "id2")
		assert.Equal(cert1.GetIssuingCA(), pem.RootCertificate("id1id2"))

		// cached
		cert2, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.Equal(cert1, cert2)

		// cached, but pubIssuer is removed
		cm.validatingIssuer = cm.signingIssuer
		cert3, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotEqual(cert1, cert3)
		assert.Equal(cert3.keyIssuerID, "id1")
		assert.Equal(cert3.pubIssuerID, "id1")
		assert.Equal(cert3.GetIssuingCA(), pem.RootCertificate("id1"))

		// cached, but keyIssuer is old
		cm.signingIssuer = &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2"}}
		cert4, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotEqual(cert3, cert4)
		assert.Equal(cert4.keyIssuerID, "id2")
		assert.Equal(cert4.pubIssuerID, "id1")
		assert.Equal(cert4.GetIssuingCA(), pem.RootCertificate("id2id1"))

		// cached, but pubIssuer is old
		cm.validatingIssuer = &issuer{ID: "id3", Issuer: &fakeIssuer{id: "id3"}}
		cert5, err := cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotEqual(cert4, cert5)
		assert.Equal(cert5.keyIssuerID, "id2")
		assert.Equal(cert5.pubIssuerID, "id3")
		assert.Equal(cert5.GetIssuingCA(), pem.RootCertificate("id2id3"))
	})

	t.Run("bad issuers", func(t *testing.T) {
		cm := &Manager{
			// The root certificate signing all newly issued certificates
			signingIssuer:    &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1", err: true}},
			validatingIssuer: &issuer{ID: "id2", Issuer: &fakeIssuer{id: "id2", err: true}},
		}

		// bad private key
		cert, err := cm.IssueCertificate(cn, time.Minute)
		assert.Nil(cert)
		assert.EqualError(err, "id1 failed")

		// bad public key
		cm.signingIssuer = &issuer{ID: "id3", Issuer: &fakeIssuer{id: "id3"}}
		cert, err = cm.IssueCertificate(cn, time.Minute)
		assert.Nil(cert)
		assert.EqualError(err, "id2 failed")

		// insert a cached cert
		cm.validatingIssuer = cm.signingIssuer
		cert, err = cm.IssueCertificate(cn, time.Minute)
		assert.NoError(err)
		assert.NotNil(cert)

		// bad public key on an existing cached cert, because the pubIssuer is new
		cm.validatingIssuer = &issuer{ID: "id1", Issuer: &fakeIssuer{id: "id1", err: true}}
		cert, err = cm.IssueCertificate(cn, time.Minute)
		assert.EqualError(err, "id1 failed")
		assert.Nil(cert)
	})
}

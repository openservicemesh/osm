package certificate

import (
	"fmt"
	"testing"
	time "time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/messaging"
)

var _ = Describe("Test Tresor Debugger", func() {
	Context("test ListIssuedCertificates()", func() {
		// Setup:
		//   1. Create a new (fake) certificate
		//   2. Reuse the same certificate as the Issuing CA
		//   3. Populate the CertManager's cache w/ cert
		cert := &Certificate{CommonName: "fake-cert-for-debugging"}
		cm := &Manager{}
		cm.cache.Store("foo", cert)

		It("lists all issued certificates", func() {
			actual := cm.ListIssuedCertificates()
			expected := []*Certificate{cert}
			Expect(actual).To(Equal(expected))
		})
	})
})

var _ = Describe("Test Certificate Manager", func() {
	defer GinkgoRecover()
	const serviceFQDN = "a.b.c"

	Context("Test Getting a certificate from the cache", func() {
		m, newCertError := NewManager(
			&fakeMRCClient{},
			validity,
			nil)
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(CommonName(serviceFQDN)))

			cachedCert := m.getFromCache(serviceFQDN)
			Expect(cachedCert).To(Equal(cert))
		})
	})
})

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

	start := time.Now()
	// Wait for one certificate rotation to be announced and terminate
	<-certRotateChan

	fmt.Printf("It took %+v to rotate certificate %s\n", time.Since(start), cn)
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

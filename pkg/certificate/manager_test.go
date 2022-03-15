package certificate

import (
	"testing"
	time "time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"
)

var (
	caCert = &Certificate{
		CommonName: "Test CA",
		Expiration: time.Now().Add(time.Hour * 24),
	}
)

type fakeIssuer struct{}

func (i *fakeIssuer) IssueCertificate(cn CommonName, validityPeriod time.Duration) (*Certificate, error) {
	return &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(validityPeriod),
	}, nil
}

var _ = Describe("Test Tresor Debugger", func() {
	Context("test ListIssuedCertificates()", func() {
		// Setup:
		//   1. Create a new (fake) certificate
		//   2. Reuse the same certificate as the Issuing CA
		//   3. Populate the CertManager's cache w/ cert
		cert := &Certificate{CommonName: "fake-cert-for-debugging"}
		cm := &manager{}
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
		validity := time.Hour
		m, newCertError := NewManager(
			caCert,
			&fakeIssuer{},
			validity,
			nil,
		)
		It("should get an issued certificate from the cache", func() {
			Expect(newCertError).ToNot(HaveOccurred())
			cert, issueCertificateError := m.IssueCertificate(serviceFQDN, validity)
			Expect(issueCertificateError).ToNot(HaveOccurred())
			Expect(cert.GetCommonName()).To(Equal(CommonName(serviceFQDN)))

			cachedCert, getCertificateError := m.GetCertificate(serviceFQDN)
			Expect(getCertificateError).ToNot(HaveOccurred())
			Expect(cachedCert).To(Equal(cert))
		})
	})
})

func TestReleaseCertificate(t *testing.T) {
	cn := CommonName("Test CN")
	cert := &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(1 * time.Hour),
	}

	manager := &manager{}
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
			_, err := manager.GetCertificate(tc.commonName)

			assert.ErrorIs(err, errCertNotFound)
		})
	}
}

func TestGetCertificate(t *testing.T) {
	cn := CommonName("Test Cert")
	cert := &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(1 * time.Hour),
	}

	expiredCn := CommonName("Expired Test Cert")
	expiredCert := &Certificate{
		CommonName: expiredCn,
		Expiration: time.Now().Add(-1 * time.Hour),
	}

	manager := &manager{}
	manager.cache.Store(cn, cert)
	manager.cache.Store(expiredCn, expiredCert)

	testCases := []struct {
		name                string
		commonName          CommonName
		expectedCertificate *Certificate
		expectedErr         error
	}{
		{
			name:                "cache hit",
			commonName:          cn,
			expectedCertificate: cert,
		},
		{
			name:        "cache miss",
			commonName:  CommonName("Wrong Cert"),
			expectedErr: errCertNotFound,
		},
		{
			name:        "certificate expiration",
			commonName:  expiredCn,
			expectedErr: errCertNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			c, err := manager.GetCertificate(tc.commonName)
			if tc.expectedErr != nil {
				assert.ErrorIs(err, tc.expectedErr)
				return
			}

			assert.Nil(err)
			assert.Equal(tc.expectedCertificate, c)
		})
	}
}

func TestListCertificate(t *testing.T) {
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

	manager := &manager{}
	manager.cache.Store(cn, cert)
	manager.cache.Store(anotherCn, anotherCert)

	cs, err := manager.ListCertificates()

	assert.Nil(err)
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

func TestGetRootCertificate(t *testing.T) {
	assert := tassert.New(t)

	manager := &manager{ca: caCert}

	got, err := manager.GetRootCertificate()

	assert.Nil(err)
	assert.Equal(caCert, got)
}

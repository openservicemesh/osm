package certificate

import (
	"testing"
	time "time"

	gomock "github.com/golang/mock/gomock"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/stretchr/testify/assert"
)

// func TestCertManagerRotation(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "Test Suite")
// }

// var _ = Describe("Test CertManagerRotation", func() {

// 	var (
// 		mockCtrl         *gomock.Controller
// 		mockConfigurator *configurator.MockConfigurator
// 	)

// 	mockCtrl = gomock.NewController(GinkgoT())

// 	cn := CommonName("foo")

// 	Context("Testing rotating expiring certificates", func() {

// 		validityPeriod := 1 * time.Hour
// 		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
// 		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Times(0)
// 		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

// 		certManager := tresor.NewFakeCertManager(mockConfigurator)

// 		It("determines whether a certificate has expired", func() {
// 			cert, err := provider.IssueCertificate(cn, validityPeriod)
// 			Expect(err).ToNot(HaveOccurred())
// 			actual := manager.ShouldRotate(cert)
// 			Expect(actual).To(BeFalse())
// 		})
// 	})

// 	Context("Testing rotating expiring certificates", func() {

// 		validityPeriod := -1 * time.Hour // negative time means this cert has already expired -- will be rotated asap

// 		mockConfigurator = configurator.NewMockConfigurator(mockCtrl)
// 		mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()
// 		mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()

// 		stop := make(chan struct{})
// 		defer close(stop)
// 		msgBroker := messaging.NewBroker(stop)
// 		certManager := tresor.NewFakeCertManager(mockConfigurator, msgBroker)

// 		certA, err := certManager.IssueCertificate(cn, validityPeriod)

// 		certRotateChan := msgBroker.GetCertPubSub().Sub(announcements.CertificateRotated.String())

// 		It("issued a new certificate", func() {
// 			Expect(err).ToNot(HaveOccurred())
// 		})

// 		It("will determine that the certificate needs to be rotated because it has already expired due to negative validity period", func() {
// 			actual := certManager.ShouldRotate(certA)
// 			Expect(actual).To(BeTrue())
// 		})

// 		It("rotates certificate", func() {
// 			done := make(chan interface{})

// 			start := time.Now()
// 			certManager.New(certManager).Start(360 * time.Second)
// 			// Wait for one certificate rotation to be announced and terminate
// 			<-certRotateChan
// 			close(done)

// 			fmt.Printf("It took %+v to rotate certificate %s\n", time.Since(start), cn)

// 			newCert, err := certManager.IssueCertificate(cn, validityPeriod)
// 			Expect(err).ToNot(HaveOccurred())
// 			Expect(newCert.GetExpiration()).ToNot(Equal(certA.GetExpiration()))
// 			Expect(newCert).ToNot(Equal(certA))
// 		})
// 	})

// })

///tresor debugger test
// var _ = Describe("Test Tresor Debugger", func() {
// 	Context("test ListIssuedCertificates()", func() {
// 		// Setup:
// 		//   1. Create a new (fake) certificate
// 		//   2. Reuse the same certificate as the Issuing CA
// 		//   3. Populate the CertManager's cache w/ cert
// 		cert := NewFakeCertificate()
// 		cm := CertManager{}
// 		cm.cache.Store("foo", cert)

// 		It("lists all issued certificates", func() {
// 			actual := cm.ListIssuedCertificates()
// 			expected := []*certificate.Certificate{cert}
// 			Expect(actual).To(Equal(expected))
// 		})
// 	})
// })

func TestReleaseCertificate(t *testing.T) {
	cn := CommonName("Test CN")
	cert := &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(1 * time.Hour),
	}

	manager := &CertManager{}
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
			assert := assert.New(t)

			manager.ReleaseCertificate(tc.commonName)
			_, err := manager.GetCertificate(tc.commonName)

			assert.ErrorIs(err, errCertNotFound)
		})
	}
}

func TestRotateCertificate(t *testing.T) {
	validity := 1 * time.Hour
	keySize := 2048
	manager := CertManager{}

	cn := CommonName("Test Cert")

	oldCert := &Certificate{
		CommonName: cn,
		Expiration: time.Now().Add(-1 * time.Hour),
	}

	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(validity).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(keySize).AnyTimes()

	stop := make(chan struct{})
	defer close(stop)

	testCases := []struct {
		name           string
		commonName     CommonName
		expectedErrMsg string
	}{
		{
			name:           "non-existing cert",
			commonName:     CommonName("Wrong Cert"),
			expectedErrMsg: "Old certificate does not exist for CN=Wrong Cert",
		},
		{
			name:       "existing cert",
			commonName: cn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			newCert, err := manager.RotateCertificate(tc.commonName)
			if len(tc.expectedErrMsg) != 0 {
				assert.EqualError(err, tc.expectedErrMsg)
				return
			}

			assert.Nil(err)
			assert.Equal(tc.commonName, newCert.GetCommonName())
			assert.True(newCert.GetExpiration().After(oldCert.GetExpiration()))

			cert, err := manager.GetCertificate(tc.commonName)
			assert.Nil(err)
			assert.Equal(newCert, cert)
		})
	}
}

func TestListCertificate(t *testing.T) {
	assert := assert.New(t)

	cn := CommonName("Test Cert")
	cert := &Certificate{
		CommonName: cn,
	}

	anotherCn := CommonName("Another Test Cert")
	anotherCert := &Certificate{
		CommonName: anotherCn,
	}

	expectedCertificates := []*Certificate{cert, anotherCert}

	manager := &CertManager{}
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

	manager := &CertManager{}
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
			assert := assert.New(t)

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

// func TestGetRootCertificate(t *testing.T) {
// 	assert := assert.New(t)

// 	validity := 1 * time.Hour

// ca := CommonName("Test CA")
// rootCertCountry := "US"
// rootCertLocality := "CA"
// rootCertOrganization := "Open Service Mesh"

// rootCert, err := NewCA(ca, validity, rootCertCountry, rootCertLocality, rootCertOrganization)
// if err != nil {
// 	t.Fatalf("Error loading CA from files %s and %s: %s", rootCertPem, rootKeyPem, err)
// }

// manager := &CertManager{}
// p, err  := NewProvider(rootCert, rootCertOrganization, 2048)
// if err != nil {
// 	t.Fatalf("error creating provider")
// }
// p.IssueCertificate(rootCert, 1 * time.Hour)

// got, err := manager.GetRootCertificate()

// assert.Nil(err)
// assert.Equal(rootCert, got)
// }

// NewFakeCertificate is a helper creating Certificates for unit tests.
func NewFakeCertificate() *Certificate {
	return &Certificate{
		PrivateKey:   pem.PrivateKey("yy"),
		CertChain:    pem.Certificate("xx"),
		IssuingCA:    pem.RootCertificate("xx"),
		Expiration:   time.Now(),
		CommonName:   "foo.bar.co.uk",
		SerialNumber: "-the-certificate-serial-number-",
	}
}

package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

func TestNewCA(t *testing.T) {
	assert := tassert.New(t)
	rootCertCountry := "US"
	rootCertLocality := "CA"

	cert, err := NewCA("Tresor CA for Testing", 2*time.Second, rootCertCountry, rootCertLocality, rootCertOrganization)
	assert.Nil(err)

	x509Cert, err := certificate.DecodePEMCertificate(cert.GetCertificateChain())
	assert.Nil(err)

	assert.Equal(2*time.Second, x509Cert.NotAfter.Sub(x509Cert.NotBefore))
	assert.Equal(x509.KeyUsageCertSign|x509.KeyUsageCRLSign, x509Cert.KeyUsage)
	assert.True(x509Cert.IsCA)
}

func TestNewCertificateFromPEM(t *testing.T) {
	assert := tassert.New(t)
	cn := certificate.CommonName("Test CA")
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := "Root Cert Organization"

	notBefore := time.Now()
	notAfter := notBefore.Add(1 * time.Hour)
	serialNumber := big.NewInt(1)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn.String(),
			Country:      []string{rootCertCountry},
			Locality:     []string{rootCertLocality},
			Organization: []string{rootCertOrganization},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(err)

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	assert.Nil(err)

	pemCert, err := certificate.EncodeCertDERtoPEM(derBytes)
	assert.Nil(err)

	pemKey, err := certificate.EncodeKeyDERtoPEM(rsaKey)
	assert.Nil(err)

	tests := []struct {
		name                   string
		pemCert                pem.Certificate
		pemKey                 pem.PrivateKey
		expectedBeforeAfterDif time.Duration
		expectedExpiration     time.Time
		expectedKeyUsage       x509.KeyUsage
		expectedIsCA           bool
		expectedCN             certificate.CommonName
		expectedErr            bool
	}{
		{
			name:                   "valid pem cert and pem key",
			pemCert:                pemCert,
			pemKey:                 pemKey,
			expectedBeforeAfterDif: 1 * time.Hour,
			expectedExpiration:     notAfter.UTC().Truncate(time.Second), // when the certificate is created time is convert to UTC and truncated
			expectedKeyUsage:       x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			expectedIsCA:           true,
			expectedCN:             cn,
			expectedErr:            false,
		},
		{
			name:        "invalid pem cert and pem key",
			pemCert:     []byte(""),
			pemKey:      []byte(""),
			expectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewCertificateFromPEM(test.pemCert, test.pemKey)
			assert.Equal(test.expectedErr, err != nil)

			if !test.expectedErr {
				assert.Equal(test.expectedCN, c.GetCommonName())
				assert.Equal(test.expectedExpiration, c.GetExpiration())

				x509Cert, err := certificate.DecodePEMCertificate(c.GetCertificateChain())
				assert.Nil(err)
				assert.Equal(test.expectedExpiration, x509Cert.NotAfter)
				assert.Equal(test.expectedBeforeAfterDif, x509Cert.NotAfter.Sub(x509Cert.NotBefore))
				assert.Equal(test.expectedKeyUsage, x509Cert.KeyUsage)
				assert.Equal(test.expectedIsCA, x509Cert.IsCA)
				assert.Equal(test.expectedCN.String(), x509Cert.Subject.CommonName)
			}
		})
	}
}

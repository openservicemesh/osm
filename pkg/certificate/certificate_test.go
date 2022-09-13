package certificate

import (
	"crypto/x509"
	"testing"
	time "time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

func TestNewFromPEM(t *testing.T) {
	assert := tassert.New(t)

	cn := CommonName("Test CA")
	notBefore := time.Now()
	notAfter := notBefore.Add(1 * time.Hour)
	pemCert, pemKey, err := CreateValidCertAndKey(notBefore, notAfter)
	assert.Nil(err)

	tests := []struct {
		name                   string
		pemCert                pem.Certificate
		pemKey                 pem.PrivateKey
		expectedBeforeAfterDif time.Duration
		expectedExpiration     time.Time
		expectedKeyUsage       x509.KeyUsage
		expectedIsCA           bool
		expectedCN             CommonName
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
		{
			name:                   "valid pem cert and invalid pem key",
			pemCert:                pemCert,
			pemKey:                 []byte(""),
			expectedBeforeAfterDif: 1 * time.Hour,
			expectedExpiration:     notAfter.UTC().Truncate(time.Second), // when the certificate is created time is convert to UTC and truncated
			expectedKeyUsage:       x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			expectedIsCA:           true,
			expectedCN:             cn,
			expectedErr:            false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := NewFromPEM(test.pemCert, test.pemKey)
			assert.Equal(test.expectedErr, err != nil)

			if !test.expectedErr {
				assert.Equal(test.expectedCN, c.GetCommonName())
				assert.Equal(test.expectedExpiration, c.GetExpiration())

				x509Cert, err := DecodePEMCertificate(c.GetCertificateChain())
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

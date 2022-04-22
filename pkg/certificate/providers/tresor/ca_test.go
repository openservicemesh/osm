package tresor

import (
	"crypto/x509"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
)

func TestNewCA(t *testing.T) {
	assert := tassert.New(t)
	rootCertCountry := "US"
	rootCertLocality := "CA"
	rootCertOrganization := testCertOrgName

	cert, err := NewCA("Tresor CA for Testing", 2*time.Second, rootCertCountry, rootCertLocality, rootCertOrganization)
	assert.Nil(err)

	x509Cert, err := certificate.DecodePEMCertificate(cert.GetCertificateChain())
	assert.Nil(err)

	assert.Equal(2*time.Second, x509Cert.NotAfter.Sub(x509Cert.NotBefore))
	assert.Equal(x509.KeyUsageCertSign|x509.KeyUsageCRLSign, x509Cert.KeyUsage)
	assert.True(x509Cert.IsCA)
}

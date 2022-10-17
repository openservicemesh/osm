package debugger

import (
	"net/http/httptest"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
)

// Tests getCertificateHandler through HTTP handler returns a certificate stringified
func TestGetCertHandler(t *testing.T) {
	assert := tassert.New(t)

	ds := DebugConfig{
		certDebugger: tresorFake.NewFake(time.Hour),
	}

	_, err := ds.certDebugger.IssueCertificate(certificate.ForServiceIdentity("commonName"))
	assert.Nil(err)

	handler := ds.getCertHandler()

	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, nil)

	actualResponseBody := responseRecorder.Body.String()

	assert.Contains(actualResponseBody, "Common Name")
	assert.Contains(actualResponseBody, "Valid Until")
	assert.Contains(actualResponseBody, "Cert Chain (SHA256)")
	assert.Contains(actualResponseBody, "x509.SignatureAlgorithm")
	assert.Contains(actualResponseBody, "x509.PublicKeyAlgorithm")
	assert.Contains(actualResponseBody, "x509.SerialNumber")
}

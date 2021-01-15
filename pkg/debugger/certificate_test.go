package debugger

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
)

// Tests getCertificateHandler through HTTP handler returns a certificate stringified
func TestGetCertHandler(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)
	mock := NewMockCertificateManagerDebugger(mockCtrl)

	ds := DebugConfig{
		certDebugger: mock,
	}

	testCert, err := tresor.NewCA("commonName", 1*time.Hour, "Country", "Locale", "Org")
	assert.Nil(err)

	// mock expected cert
	mock.EXPECT().ListIssuedCertificates().Return([]certificate.Certificater{
		testCert,
	})

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

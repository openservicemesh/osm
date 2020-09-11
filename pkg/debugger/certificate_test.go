package debugger

import (
	"net/http/httptest"
	"time"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
)

// Tests if namespace handler returns default namespace correctly
var _ = Describe("Test debugger certificate methods", func() {
	var (
		mockCtrl *gomock.Controller
		mock     *MockCertificateManagerDebugger
	)
	mockCtrl = gomock.NewController(GinkgoT())

	BeforeEach(func() {
		var err error
		mock = NewMockCertificateManagerDebugger(mockCtrl)
		Expect(err).To(BeNil())
	})

	It("returns stringyfied list of certificates", func() {
		ds := debugServer{
			certDebugger: mock,
		}

		testCert, err := tresor.NewCA("commonName", 1*time.Hour, "Country", "Locale", "Org")
		Expect(err).To(BeNil())

		// mock expected cert
		mock.EXPECT().ListIssuedCertificates().Return([]certificate.Certificater{
			testCert,
		})

		handler := ds.getCertHandler()

		responseRecorder := httptest.NewRecorder()
		handler.ServeHTTP(responseRecorder, nil)

		actualResponseBody := responseRecorder.Body.String()
		// Expect some of the string format types printed, but not checking values
		Expect(actualResponseBody).To(ContainSubstring("Common Name"))
		Expect(actualResponseBody).To(ContainSubstring("Valid Until"))
		Expect(actualResponseBody).To(ContainSubstring("Cert Chain (SHA256)"))
		Expect(actualResponseBody).To(ContainSubstring("x509.SignatureAlgorithm"))
		Expect(actualResponseBody).To(ContainSubstring("x509.PublicKeyAlgorithm"))
		Expect(actualResponseBody).To(ContainSubstring("x509.SerialNumber"))
	})
})

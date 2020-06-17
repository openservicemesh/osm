package debugger

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}

var _ = Describe("Test debugger method", func() {

	Context("Testing GetPolicy", func() {
		It("return policy", func() {
			ds := debugServer{}
			smiPoliciesHandler := ds.getSMIPoliciesHandler()
			responseRecorder := httptest.NewRecorder()
			smiPoliciesHandler.ServeHTTP(responseRecorder, nil)
			actualResponseBody := responseRecorder.Body.String()
			expectedResponseBody := "This here is what we expect the body of the HTTP response to be..."
			Expect(actualResponseBody).To(Equal(expectedResponseBody))
		})
	})
})

package debugger

import (
	"fmt"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

)

// Tests if namespace handler returns default namespace correctly
var _ = Describe("Test debugger methods", func() {
	Context("Testing getMonitoredNamespacesHandler()", func() {
		It("returns JSON serialized monitored namespaces", func() {
			mc := NewFakeMeshCatalogDebugger()
			ds := debugServer{
				meshCatalogDebugger: mc,
			}
			monitoredNamespacesHandler := ds.getMonitoredNamespacesHandler()
			responseRecorder := httptest.NewRecorder()
			monitoredNamespacesHandler.ServeHTTP(responseRecorder, nil)
			actualResponseBody := responseRecorder.Body.String()
			expectedResponseBody := `{"namespaces":["default"]}`
			Expect(actualResponseBody).To(Equal(expectedResponseBody), fmt.Sprintf("Actual value did not match expectations:\n%s", actualResponseBody))
		})
	})
})

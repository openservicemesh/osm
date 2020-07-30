package debugger

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/namespace"
)

func TestEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}

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
			expectedResponseBody := `{default}`
			Expect(actualResponseBody).To(Equal(expectedResponseBody), fmt.Sprintf("Actual value did not match expectations:\n%s", actualResponseBody))
		})
	})
})

type fakeMeshCatalogDebuger struct{}

// ListMonitoredNamespaces implements MeshCatalogDebugger
func (f fakeMeshCatalogDebuger) ListMonitoredNamespaces() ([]string) {
	return []string{tests.Namespace}
}

// NewFakeMeshCatalogDebugger implements and creates a new MeshCatalogDebugger
func NewFakeMeshCatalogDebugger() MeshCatalogDebugger {
	return fakeMeshCatalogDebuger{}
}

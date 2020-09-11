package debugger

import (
	"fmt"
	"net/http/httptest"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests if namespace handler returns default namespace correctly
var _ = Describe("Test debugger methods", func() {
	var (
		mockCtrl *gomock.Controller
		mock     *MockMeshCatalogDebugger
	)
	mockCtrl = gomock.NewController(GinkgoT())

	BeforeEach(func() {
		mock = NewMockMeshCatalogDebugger(mockCtrl)
	})

	It("returns JSON serialized monitored namespaces", func() {
		ds := debugServer{
			meshCatalogDebugger: mock,
		}
		monitoredNamespacesHandler := ds.getMonitoredNamespacesHandler()

		uniqueNs := tests.GetUnique([]string{
			tests.BookbuyerService.Namespace, // default
			tests.BookstoreService.Namespace, // default
		})

		mock.EXPECT().ListMonitoredNamespaces().Return(uniqueNs)

		responseRecorder := httptest.NewRecorder()
		monitoredNamespacesHandler.ServeHTTP(responseRecorder, nil)
		actualResponseBody := responseRecorder.Body.String()
		expectedResponseBody := `{"namespaces":["default"]}`
		Expect(actualResponseBody).To(Equal(expectedResponseBody), fmt.Sprintf("Actual value did not match expectations:\n%s", actualResponseBody))
	})

})

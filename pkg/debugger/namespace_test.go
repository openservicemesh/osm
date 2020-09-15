package debugger

import (
	"net/http/httptest"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/tests"
)

func TestMonitoredNamespaceHandler(t *testing.T) {
	assert := assert.New(t)
	mockCtrl := gomock.NewController(t)
	mock := NewMockMeshCatalogDebugger(mockCtrl)

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
	assert.Equal(expectedResponseBody, actualResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}

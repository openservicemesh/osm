package debugger

import (
	"net/http/httptest"
	"testing"

	"github.com/openservicemesh/osm/pkg/k8s"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests getMonitoredNamespaces through HTTP handler returns a the list of monitored namespaces
func TestMonitoredNamespaceHandler(t *testing.T) {
	assert := tassert.New(t)

	mockKubeController := k8s.NewMockController(gomock.NewController(t))

	ds := DebugConfig{
		kubeController: mockKubeController,
	}
	monitoredNamespacesHandler := ds.getMonitoredNamespacesHandler()

	uniqueNs := tests.GetUnique([]string{
		tests.BookbuyerService.Namespace,   // default
		tests.BookstoreV1Service.Namespace, // default
	})

	mockKubeController.EXPECT().ListMonitoredNamespaces().Return(uniqueNs, nil)

	responseRecorder := httptest.NewRecorder()
	monitoredNamespacesHandler.ServeHTTP(responseRecorder, nil)
	actualResponseBody := responseRecorder.Body.String()
	expectedResponseBody := `{"namespaces":["default"]}`
	assert.Equal(expectedResponseBody, actualResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}

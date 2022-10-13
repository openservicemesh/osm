package debugger

import (
	"net/http/httptest"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/messaging"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/tests"
)

// Tests getMonitoredNamespaces through HTTP handler returns a the list of monitored namespaces
func TestMonitoredNamespaceHandler(t *testing.T) {
	assert := tassert.New(t)

	mock := compute.NewMockInterface(gomock.NewController(t))
	stop := make(chan struct{})
	meshCatalog := catalog.NewMeshCatalog(
		mock,
		tresorFake.NewFake(time.Hour),
		stop,
		messaging.NewBroker(stop),
	)

	ds := DebugConfig{
		meshCatalog: meshCatalog,
	}
	monitoredNamespacesHandler := ds.getMonitoredNamespacesHandler()

	uniqueNs := tests.GetUnique([]string{
		tests.BookbuyerService.Namespace,   // default
		tests.BookstoreV1Service.Namespace, // default
	})

	mock.EXPECT().ListNamespaces().Return(uniqueNs, nil)

	responseRecorder := httptest.NewRecorder()
	monitoredNamespacesHandler.ServeHTTP(responseRecorder, nil)
	actualResponseBody := responseRecorder.Body.String()
	expectedResponseBody := `{"namespaces":["default"]}`
	assert.Equal(expectedResponseBody, actualResponseBody, "Actual value did not match expectations:\n%s", actualResponseBody)
}

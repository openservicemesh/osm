package debugger

import (
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

// Tests GetHandlers returns the expected debug endpoints and non-nil handlers
func TestGetHandlers(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)

	mockCertDebugger := NewMockCertificateManagerDebugger(mockCtrl)
	mockXdsDebugger := NewMockXDSDebugger(mockCtrl)
	mockCatalogDebugger := NewMockMeshCatalogDebugger(mockCtrl)
	mockConfig := configurator.NewMockConfigurator(mockCtrl)
	client := testclient.NewSimpleClientset()
	mockKubeController := k8s.NewMockController(mockCtrl)

	ds := NewDebugConfig(mockCertDebugger,
		mockXdsDebugger,
		mockCatalogDebugger,
		nil,
		client,
		mockConfig,
		mockKubeController)

	handlers := ds.GetHandlers()

	debugEndpoints := []string{
		"/debug/certs",
		"/debug/xds",
		"/debug/proxy",
		"/debug/policies",
		"/debug/config",
		"/debug/namespaces",
		// Pprof handlers
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/profile",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
	}

	for _, endpoint := range debugEndpoints {
		handler, found := handlers[endpoint]
		assert.True(found)
		assert.NotNil(handler)
	}
}

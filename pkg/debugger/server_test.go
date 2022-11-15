package debugger

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/catalog"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
)

// Tests GetHandlers returns the expected debug endpoints and non-nil handlers
func TestGetHandlers(t *testing.T) {
	assert := tassert.New(t)
	mockCtrl := gomock.NewController(t)

	cm := tresorFake.NewFake(time.Hour)
	mockXdsDebugger := NewMockXDSDebugger(mockCtrl)
	client := testclient.NewSimpleClientset()
	mockComputeInterface := catalog.NewMockInterface(mockCtrl)
	proxyRegistry := registry.NewProxyRegistry()

	ds := NewDebugConfig(cm,
		mockXdsDebugger,
		proxyRegistry,
		nil,
		client,
		mockComputeInterface,
		nil)

	handlers := ds.GetHandlers()

	debugEndpoints := []string{
		"/debug/certs",
		"/debug/xds",
		"/debug/proxy",
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

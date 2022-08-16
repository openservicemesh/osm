package registry

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	proxyRegistry := NewProxyRegistry(nil, nil)
	proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil, 1)

	It("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
	})

	Context("Test register/unregister proxies", func() {
		It("no proxies connected", func() {
			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))
		})

		It("one proxy connected to OSM", func() {
			proxyRegistry.RegisterProxy(proxy)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(1))
			Expect(connectedProxies).To(ContainElement(proxy))
		})

		It("one proxy disconnected from OSM", func() {
			proxyRegistry.UnregisterProxy(1)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))
		})
	})
})

func TestRegisterUnregister(t *testing.T) {
	assert := tassert.New(t)
	proxyRegistry := NewProxyRegistry(nil, nil)

	proxyUUID := uuid.New()
	var i int64
	for i = 0; i < 10; i++ {
		proxy := envoy.NewProxy(envoy.KindSidecar, proxyUUID, identity.New("foo", "bar"), nil, i)
		assert.Nil(proxyRegistry.GetConnectedProxy(i))
		proxyRegistry.RegisterProxy(proxy)
		assert.Equal(proxy, proxyRegistry.GetConnectedProxy(i))
	}

	assert.Equal(10, proxyRegistry.GetConnectedProxyCount())

	for i = 0; i < 10; i++ {
		proxyRegistry.UnregisterProxy(i)
		assert.Nil(proxyRegistry.GetConnectedProxy(i))
	}
	assert.Equal(0, proxyRegistry.GetConnectedProxyCount())
}

func BenchmarkRegistryAdd(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	for n := 0; n < b.N; n++ {
		proxyRegistry := NewProxyRegistry(nil, nil)
		total := 10000

		for j := 0; j < total; j++ {
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil, int64(j))
			proxyRegistry.RegisterProxy(proxy)
			proxyRegistry.UnregisterProxy(int64(j))
		}
		if proxyRegistry.GetConnectedProxyCount() != 0 {
			b.Errorf("Expected %d proxies, got %d", 0, proxyRegistry.GetConnectedProxyCount())
		}
	}
}

func BenchmarkRegistryGetCount(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	proxyRegistry := NewProxyRegistry(nil, nil)
	total := 10000

	wg := sync.WaitGroup{}
	wg.Add(total)
	for j := 0; j < total; j++ {
		go func() {
			proxyRegistry.RegisterProxy(
				envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil, 1))
			wg.Done()
		}()
	}
	wg.Wait()

	for n := 0; n < b.N; n++ {
		if proxyRegistry.GetConnectedProxyCount() != total {
			b.Errorf("Expected %d proxies, got %d", total, proxyRegistry.GetConnectedProxyCount())
		}
	}
}

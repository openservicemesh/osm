package registry

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	proxyRegistry := NewProxyRegistry(nil, nil)
	proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil)

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

			_, ok := connectedProxies[proxy.UUID.String()]
			Expect(ok).To(BeTrue())
		})

		It("one proxy disconnected from OSM", func() {
			proxyRegistry.UnregisterProxy(proxy)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))
		})
	})
})

func TestRegisterUnregister(t *testing.T) {
	if err := logger.SetLogLevel("error"); err != nil {
		t.Logf("Failed to set log level to error: %s", err)
	}

	proxyRegistry := NewProxyRegistry(nil, nil)
	total := 10000

	wg := sync.WaitGroup{}
	wg.Add(total)
	for j := 0; j < total; j++ {
		proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil)

		innerWg := sync.WaitGroup{}
		innerWg.Add(1)
		go func() {
			proxyRegistry.RegisterProxy(proxy)
			innerWg.Done()
		}()

		go func() {
			innerWg.Wait()
			proxyRegistry.UnregisterProxy(proxy)
			wg.Done()
		}()
	}

	wg.Wait()
	if proxyRegistry.GetConnectedProxyCount() != 0 {
		t.Errorf("Expected 0 proxies, got %d", proxyRegistry.GetConnectedProxyCount())
	}
}

func BenchmarkRegistryAdd(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	for n := 0; n < b.N; n++ {
		proxyRegistry := NewProxyRegistry(nil, nil)
		total := 10000

		wg := sync.WaitGroup{}
		wg.Add(total)
		for j := 0; j < total; j++ {
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil)

			innerWg := sync.WaitGroup{}
			innerWg.Add(1)
			go func() {
				proxyRegistry.RegisterProxy(proxy)
				innerWg.Done()
			}()

			go func() {
				innerWg.Wait()
				proxyRegistry.UnregisterProxy(proxy)
				wg.Done()
			}()
		}

		wg.Wait()
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
				envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil))
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

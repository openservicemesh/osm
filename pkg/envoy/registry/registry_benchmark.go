package registry

import (
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/logger"
)

func BenchmarkRegistryAdd(b *testing.B) {
	if err := logger.SetLogLevel("error"); err != nil {
		b.Logf("Failed to set log level to error: %s", err)
	}

	for n := 0; n < b.N; n++ {
		proxyRegistry := NewProxyRegistry(nil, nil)
		total := 10000

		wg := sync.WaitGroup{}
		wg.Add(total * 2)
		for j := 0; j < total; j++ {
			proxy := envoy.NewProxy(envoy.KindSidecar, uuid.New(), identity.New("foo", "bar"), nil)
			go func() {
				proxyRegistry.RegisterProxy(proxy)
				wg.Done()
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

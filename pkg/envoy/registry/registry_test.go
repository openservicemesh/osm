package registry

import (
	"sync"

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

		It("ensures the correctness of proxy count", func() {
			err := logger.SetLogLevel("error")
			Expect(err).To(BeNil())

			proxyRegistry := NewProxyRegistry(nil, nil)
			total := 10000

			wg := sync.WaitGroup{}
			wg.Add(total*2)
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
			Expect(proxyRegistry.GetConnectedProxyCount()).To(Equal(0))
		})
	})

})

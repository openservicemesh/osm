package registry

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	proxyRegistry := NewProxyRegistry()
	certCommonName := certificate.CommonName("foo")
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	Context("Test register/unregister proxies", func() {
		It("no proxies connected or disconnected", func() {
			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := proxyRegistry.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))
		})

		It("one proxy connected to OSM", func() {
			proxyRegistry.RegisterProxy(proxy)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(1))

			disconnectedProxies := proxyRegistry.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))

			_, ok := connectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})

		It("one proxy disconnected from OSM", func() {
			proxyRegistry.UnregisterProxy(proxy)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := proxyRegistry.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(1))

			_, ok := disconnectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})
	})
})

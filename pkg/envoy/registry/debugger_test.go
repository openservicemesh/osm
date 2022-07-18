package registry

import (
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	proxyRegistry := NewProxyRegistry(nil, nil)
	certCommonName := certificate.CommonName(fmt.Sprintf("%s.sidecar.foo.bar", uuid.New()))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy, err := envoy.NewProxy(certCommonName, certSerialNumber, nil)

	Context("Proxy is valid", func() {
		Expect(proxy).ToNot((BeNil()))
		Expect(err).ToNot(HaveOccurred())
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

			_, ok := connectedProxies[certCommonName]
			Expect(ok).To(BeTrue())
		})

		It("one proxy disconnected from OSM", func() {
			proxyRegistry.UnregisterProxy(proxy)

			connectedProxies := proxyRegistry.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))
		})
	})
})

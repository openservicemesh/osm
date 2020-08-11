package catalog

import (
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/envoy"
)

var _ = Describe("Test catalog proxy register/unregister", func() {
	Context("Test register/unregister proxies", func() {
		mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())
		cn := certificate.CommonName("foo")
		proxy := envoy.NewProxy(cn, nil)

		It("no proxies expected, connected or disconnected", func() {
			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))
		})

		It("expect one proxy to connect", func() {
			// mc.RegisterProxy(proxy)
			mc.ExpectProxy(cn)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(1))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))

			_, ok := expectedProxies[cn]
			Expect(ok).To(BeTrue())
		})

		It("one proxy connected to OSM", func() {
			mc.RegisterProxy(proxy)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(1))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(0))

			_, ok := connectedProxies[cn]
			Expect(ok).To(BeTrue())
		})

		It("one proxy disconnected from OSM", func() {
			mc.UnregisterProxy(proxy)

			expectedProxies := mc.ListExpectedProxies()
			Expect(len(expectedProxies)).To(Equal(0))

			connectedProxies := mc.ListConnectedProxies()
			Expect(len(connectedProxies)).To(Equal(0))

			disconnectedProxies := mc.ListDisconnectedProxies()
			Expect(len(disconnectedProxies)).To(Equal(1))

			_, ok := disconnectedProxies[cn]
			Expect(ok).To(BeTrue())
		})
	})
})

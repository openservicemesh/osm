package lds

import (
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/open-service-mesh/osm/pkg/catalog"
	"github.com/open-service-mesh/osm/pkg/service"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/envoy"
	"github.com/open-service-mesh/osm/pkg/tests"
)

var _ = Describe("Test LDS response", func() {
	Context("Test getInboundIngressFilterChain()", func() {
		It("constructs filter chain used for ingress", func() {
			marshalledConnManager, err := envoy.MessageToAny(getHTTPConnectionManager("fake"))
			Expect(err).ToNot(HaveOccurred())
			filterChain, err := getInboundIngressFilterChain(tests.BookstoreService, marshalledConnManager)
			Expect(err).ToNot(HaveOccurred())
			Expect(filterChain.FilterChainMatch.TransportProtocol).To(Equal(envoy.TransportProtocolTLS))
			Expect(len(filterChain.Filters)).To(Equal(1))
			Expect(filterChain.Filters[0].Name).To(Equal(wellknown.HTTPConnectionManager))
		})

		It("constructs in-mesh filter chain", func() {
			mc := catalog.NewFakeMeshCatalog(testclient.NewSimpleClientset())
			filterChain, err := getInboundInMeshFilterChain(tests.BookstoreService, mc, nil)
			Expect(err).ToNot(HaveOccurred())

			expectedServerNames := []string{tests.BookbuyerService.GetCommonName().String()}

			// Show what this looks like (human readable)!  And ensure this is setup correctly!
			Expect(expectedServerNames[0]).To(Equal("bookbuyer.default.svc.cluster.local"))

			Expect(filterChain.FilterChainMatch.TransportProtocol).To(Equal(envoy.TransportProtocolTLS))
			Expect(filterChain.FilterChainMatch.ServerNames).To(Equal(expectedServerNames))

			// Ensure the UpstreamTlsContext.Sni field from the client matches one of the strings
			// in the servers FilterChainMatch.ServerNames
			tlsContext := envoy.GetUpstreamTLSContext(tests.BookbuyerService)
			Expect(tlsContext.Sni).To(Equal(filterChain.FilterChainMatch.ServerNames[0]))

			// Show what that actually looks like
			Expect(tlsContext.Sni).To(Equal("bookbuyer.default.svc.cluster.local"))
		})
	})

	Context("Test toCommonNamesList()", func() {
		It("returns a list of common names", func() {
			actual := toCommonNamesList([]service.NamespacedService{tests.BookbuyerService, tests.BookstoreService})
			expected := []string{
				"bookbuyer.default.svc.cluster.local",
				"bookstore.default.svc.cluster.local",
			}
			Expect(actual).To(Equal(expected))
		})
	})
})

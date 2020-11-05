package lds

import (
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/tests"
)

var _ = Describe("Test LDS response", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())

	Context("Test getInboundIngressFilterChain()", func() {
		BeforeEach(func() {
			mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

			mockConfigurator.EXPECT().IsTracingEnabled().Return(false).AnyTimes()
			mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).AnyTimes()
			mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).AnyTimes()
		})

		It("constructs filter chain used for HTTPS ingress", func() {
			expectedServerNames := []string{tests.BookstoreV1Service.ServerName()}

			mockConfigurator.EXPECT().UseHTTPSIngress().Return(true).AnyTimes()

			filterChains := getIngressFilterChains(tests.BookstoreV1Service, mockConfigurator)
			Expect(len(filterChains)).To(Equal(2))
			for _, filterChain := range filterChains {
				Expect(filterChain.FilterChainMatch.TransportProtocol).To(Equal(envoy.TransportProtocolTLS))
				Expect(len(filterChain.Filters)).To(Equal(1))
				Expect(filterChain.Filters[0].Name).To(Equal(wellknown.HTTPConnectionManager))
			}

			// filter chain with SNI matching
			Expect(filterChains[0].FilterChainMatch.ServerNames).To(Equal(expectedServerNames))

			// filter chain without SNI matching
			Expect(filterChains[1].FilterChainMatch.ServerNames).To(BeNil())
		})

		It("constructs filter chain used for HTTP ingress", func() {
			mockConfigurator.EXPECT().UseHTTPSIngress().Return(false).AnyTimes()

			filterChains := getIngressFilterChains(tests.BookstoreV1Service, mockConfigurator)
			Expect(len(filterChains)).To(Equal(1))
			for _, filterChain := range filterChains {
				Expect(filterChain.FilterChainMatch.TransportProtocol).To(Equal(""))
				Expect(len(filterChain.Filters)).To(Equal(1))
				Expect(filterChain.Filters[0].Name).To(Equal(wellknown.HTTPConnectionManager))
			}
			Expect(len(filterChains[0].FilterChainMatch.ServerNames)).To(Equal(0)) // filter chain without SNI matching
		})

		It("constructs in-mesh filter chain", func() {
			filterChain, err := getInboundInMeshFilterChain(tests.BookstoreV1Service, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())

			expectedServerNames := []string{tests.BookstoreV1Service.ServerName()}

			// Show what this looks like (human readable)!  And ensure this is setup correctly!
			Expect(expectedServerNames[0]).To(Equal("bookstore-v1.default.svc.cluster.local"))

			Expect(filterChain.FilterChainMatch.TransportProtocol).To(Equal(envoy.TransportProtocolTLS))
			Expect(filterChain.FilterChainMatch.ServerNames).To(Equal(expectedServerNames))

			// Ensure the UpstreamTlsContext.Sni field from the client matches one of the strings
			// in the servers FilterChainMatch.ServerNames
			tlsContext := envoy.GetUpstreamTLSContext(tests.BookbuyerService, tests.BookstoreV1Service)
			Expect(tlsContext.Sni).To(Equal(filterChain.FilterChainMatch.ServerNames[0]))

			// Show what that actually looks like
			Expect(tlsContext.Sni).To(Equal("bookstore-v1.default.svc.cluster.local"))
		})
	})
})

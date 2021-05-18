package ads

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
)

var _ = Describe("Test ADS gRPC helpers", func() {
	defer GinkgoRecover()

	Context("Test recordEnvoyPodMetadata()", func() {
		request := &xds_discovery.DiscoveryRequest{}
		proxy := &envoy.Proxy{}
		proxyRegistry := registry.NewProxyRegistry()
		It("checks Service Accounts from NodeID and Cert", func() {
			actual := recordEnvoyPodMetadata(request, proxy, proxyRegistry)
			Expect(actual).To(Equal(errServiceAccountMismatch))
		})
	})
})

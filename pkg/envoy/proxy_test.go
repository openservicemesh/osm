package envoy

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	svc = "service-name"
	ns  = "some-namespace"
)

var _ = Describe("Test proxy methods", func() {
	Context("Testing proxy.GetCommonName()", func() {
		It("should return DNS-1123 CN of the proxy", func() {
			commonNameForProxy := fmt.Sprintf("UUID-of-proxy.%s.%s.one.two.three.co.uk", svc, ns)
			cn := certificate.CommonName(commonNameForProxy)
			proxy := NewProxy(cn, nil)
			actualCN := proxy.GetCommonName()
			Expect(actualCN).To(Equal(certificate.CommonName(commonNameForProxy)))
		})
	})
})

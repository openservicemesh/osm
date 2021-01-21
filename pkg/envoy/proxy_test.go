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
	certCommonName := certificate.CommonName(fmt.Sprintf("UUID-of-proxy.%s.%s.one.two.three.co.uk", svc, ns))
	certSerialNumber := certificate.SerialNumber("123456")
	proxy := NewProxy(certCommonName, certSerialNumber, nil)

	Context("Testing proxy.GetCertificateCommonName()", func() {
		It("should return DNS-1123 CN of the proxy", func() {
			actualCN := proxy.GetCertificateCommonName()
			Expect(actualCN).To(Equal(certCommonName))
		})
	})

	Context("Testing proxy.GetCertificateSerialNumber()", func() {
		It("should return certificate serial number", func() {
			actualSerialNumber := proxy.GetCertificateSerialNumber()
			Expect(actualSerialNumber).To(Equal(certSerialNumber))
		})
	})
})

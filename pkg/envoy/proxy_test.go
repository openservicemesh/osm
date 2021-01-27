package envoy

import (
	"fmt"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	svc = "service-name"
	ns  = "some-namespace"
)

var _ = Describe("Test proxy methods", func() {
	certCommonName := certificate.CommonName(fmt.Sprintf("UUID-of-proxy1234566623211353.%s.%s.one.two.three.co.uk", svc, ns))
	certSerialNumber := certificate.SerialNumber("123456")
	podUID := uuid.New().String()
	proxy := NewProxy(certCommonName, certSerialNumber, nil)

	Context("test GetPodUID() with empty Pod Metadata field", func() {
		It("returns correct values", func() {
			Expect(proxy.GetPodUID()).To(Equal(""))
		})
	})

	Context("test correctness proxy object creation", func() {
		It("returns correct values", func() {
			Expect(proxy.GetCertificateCommonName()).To(Equal(certCommonName))
			Expect(proxy.GetCertificateSerialNumber()).To(Equal(certSerialNumber))

			proxy.PodMetadata = &PodMetadata{
				UID: podUID,
			}
			Expect(proxy.GetPodUID()).To(Equal(podUID))
		})
	})
})

package tresor

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func firstLine(str string) string {
	return strings.Split(str, "\n")[0]
}

var _ = Describe("Test Certificate", func() {
	Context("Test creation of self-signed certificates", func() {
		cert, pk, err := NewSelfSignedCert("host", "org", 3*time.Second)
		It("should have created a cert", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(firstLine(string(cert))).To(Equal("-----BEGIN CERTIFICATE-----"))
			Expect(firstLine(string(pk))).To(Equal("-----BEGIN PRIVATE KEY-----"))
		})
	})
})

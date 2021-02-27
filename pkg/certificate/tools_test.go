package certificate

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
)

var _ = Describe("Test XDS certificate tooling", func() {

	Context("Test getCertificateCommonNameMeta()", func() {
		It("parses CN into certificateCommonNameMeta", func() {
			proxyUUID := uuid.New()
			testNamespace := uuid.New().String()
			serviceAccount := uuid.New().String()

			cn := CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, serviceAccount, testNamespace))

			cnMeta, err := getCertificateCommonNameMeta(cn)
			Expect(err).ToNot(HaveOccurred())

			expected := &certificateCommonNameMeta{
				ProxyUUID:      proxyUUID,
				ServiceAccount: serviceAccount,
				Namespace:      testNamespace,
			}
			Expect(cnMeta).To(Equal(expected))
		})

		It("parses CN into certificateCommonNameMeta", func() {
			_, err := getCertificateCommonNameMeta("a")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Test NewCertCommonNameWithProxyID() and getCertificateCommonNameMeta() together", func() {
		It("returns the the CommonName of the form <proxyID>.<namespace>", func() {

			proxyUUID := uuid.New()
			serviceAccount := uuid.New().String()
			namespace := uuid.New().String()

			cn := CommonName(fmt.Sprintf("%s.%s.%s", proxyUUID, serviceAccount, namespace))

			actualMeta, err := getCertificateCommonNameMeta(cn)
			expectedMeta := certificateCommonNameMeta{
				ProxyUUID:      proxyUUID,
				ServiceAccount: serviceAccount,
				Namespace:      namespace,
			}
			Expect(err).ToNot(HaveOccurred())
			Expect(actualMeta).To(Equal(&expectedMeta))
		})
	})

})

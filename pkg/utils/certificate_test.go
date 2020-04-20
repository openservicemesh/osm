package utils

import (
	"fmt"

	"github.com/open-service-mesh/osm/pkg/certificate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	testServiceAccountName = "test-service-account"
	testNamespace          = "test-namespace"
	testSubDomain          = "foo.test.mesh"
)

var _ = Describe("Testing utils helpers", func() {
	Context("Test NewCertCommonNameWithUUID", func() {
		It("Should return the the CommonName of the form <uuid>;<service_account_name>.<namespace>.<sub_domain>.", func() {
			cn := NewCertCommonNameWithUUID(testServiceAccountName, testNamespace, testSubDomain)
			cnMeta := GetCertificateCommonNameMeta(cn)
			expected := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", cnMeta.UUID, cnMeta.ServiceAccountName, cnMeta.Namespace, cnMeta.SubDomain))
			Expect(cn).To(Equal(expected))
		})
	})
})

var _ = Describe("Testing utils helpers", func() {
	Context("Test GetCertificateCommonNameMeta", func() {
		It("Should return the the Certificate's Common Name metadata correctly.", func() {
			cn := NewCertCommonNameWithUUID(testServiceAccountName, testNamespace, testSubDomain)
			cnMeta := GetCertificateCommonNameMeta(cn)
			Expect(cnMeta.ServiceAccountName).To(Equal(testServiceAccountName))
			Expect(IsValidUUID(cnMeta.UUID)).To(BeTrue())
			Expect(cnMeta.Namespace).To(Equal(testNamespace))
			Expect(cnMeta.SubDomain).To(Equal(testSubDomain))
			expected := certificate.CommonName(fmt.Sprintf("%s.%s.%s.%s", cnMeta.UUID, cnMeta.ServiceAccountName, cnMeta.Namespace, cnMeta.SubDomain))
			Expect(cn).To(Equal(expected))
		})

		It("Should panic because Certificate's Common Name doesn't have the correct format.", func() {
			CausePanic := func() {
				invalidCN := certificate.CommonName("ab.c")
				GetCertificateCommonNameMeta(invalidCN)
			}

			Expect(CausePanic).To(Panic())
		})
	})
})

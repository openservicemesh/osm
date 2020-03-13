package utils

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/constants"
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
			Expect(cn).To(Equal(fmt.Sprintf("%s%s%s.%s.%s", cnMeta.UUID, constants.CertCommonNameUUIDServiceDelimiter, cnMeta.ServiceAccountName, cnMeta.Namespace, cnMeta.SubDomain)))
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
			Expect(cn).To(Equal(fmt.Sprintf("%s%s%s.%s.%s", cnMeta.UUID, constants.CertCommonNameUUIDServiceDelimiter, cnMeta.ServiceAccountName, cnMeta.Namespace, cnMeta.SubDomain)))
		})

		It("Should panic because Certificate's Common Name doesn't have the correct format.", func() {
			CausePanic := func() {
				invalidCN := "ab.c"
				GetCertificateCommonNameMeta(invalidCN)
			}

			Expect(CausePanic).To(Panic())
		})
	})
})

var _ = Describe("Testing utils helpers", func() {
	Context("Test GetCertCommonNameWithoutUUID", func() {
		It("Should return the the CommonName without UUID if present.", func() {
			cnWithoutUUID := "svca.ns.osm.mesh"
			cnWithUUID := fmt.Sprintf("4507ae67-a401-404e-aa93-cc5ee3edfd6e%s%s", constants.CertCommonNameUUIDServiceDelimiter, cnWithoutUUID)
			cn := GetCertCommonNameWithoutUUID(cnWithUUID)
			Expect(cn).To(Equal(cnWithoutUUID))
			cn = GetCertCommonNameWithoutUUID(cnWithoutUUID)
			Expect(cn).To(Equal(cnWithoutUUID))
		})
	})
})

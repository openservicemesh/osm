package service

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test types helpers", func() {
	Context("Tests namespace unmarshalling", func() {
		namespace := "randomNamespace"
		serviceName := "randomServiceName"

		It("Interface marshals and unmarshals preserving the exact same data", func() {
			svn := NamespacedService{
				Namespace: namespace,
				Service:   serviceName,
			}

			str := svn.String()
			svn2, err := UnmarshalNamespacedService(str)

			Expect(err).ToNot(HaveOccurred())
			Expect(*svn2).To(Equal(svn))
		})

		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("/svnc")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("svnc/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("/svnc/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalNamespacedService("test")
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Test CertName interface", func() {
		It("Interface marshals and unmarshals preserving the exact same data", func() {
			InitialObj := CertResource{
				CertType: ServiceCertType,
				Service: NamespacedService{
					Namespace: "test-namespace",
					Service:   "test-service",
				},
			}

			// Marshal/stringify it
			marshaledStr := InitialObj.String()

			// Unmarshal it back from the string
			finalObj, _ := UnmarshalCertResource(marshaledStr)

			// First and final object must be equal
			Expect(*finalObj).To(Equal(InitialObj))
		})
	})

	Context("Test getRequestedCertType()", func() {
		It("returns service cert", func() {
			actual, err := UnmarshalCertResource("service-cert:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(ServiceCertType))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})
		It("returns root cert for mTLS", func() {
			actual, err := UnmarshalCertResource("root-cert-for-mtls-outbound:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForMTLSOutbound))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})

		It("returns root cert for non-mTLS", func() {
			actual, err := UnmarshalCertResource("root-cert-https:namespace-test/blahBlahBlahCert")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual.CertType).To(Equal(RootCertTypeForHTTPS))
			Expect(actual.Service.Namespace).To(Equal("namespace-test"))
			Expect(actual.Service.Service).To(Equal("blahBlahBlahCert"))
		})

		It("returns an error (invalid formatting)", func() {
			_, err := UnmarshalCertResource("blahBlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid formatting)", func() {
			_, err := UnmarshalCertResource("blahBlahBlahCert:moreblabla/amazingservice:bla")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (missing cert type)", func() {
			_, err := UnmarshalCertResource("blahBlahBlahCert/service")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (service is not namespaced)", func() {
			_, err := UnmarshalCertResource("root-cert-https:blahBlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid namespace formatting)", func() {
			_, err := UnmarshalCertResource("root-cert-https:blah/BlahBl/ahCert")
			Expect(err).To(HaveOccurred())
		})
		It("returns an error (empty left-side namespace)", func() {
			_, err := UnmarshalCertResource("root-cert-https:/ahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (empty cert type)", func() {
			_, err := UnmarshalCertResource(":ns/svc")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (empty slice on right/wrong number of slices)", func() {
			_, err := UnmarshalCertResource("root-cert-https:aaa/ahCert:")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid serv type)", func() {
			_, err := UnmarshalCertResource("revoked-cert:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})

		It("returns an error (invalid mtls cert type)", func() {
			_, err := UnmarshalCertResource("oot-cert-for-mtls-diagonalstream:blah/BlahBlahCert")
			Expect(err).To(HaveOccurred())
		})
	})

})

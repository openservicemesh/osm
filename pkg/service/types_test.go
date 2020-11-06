package service

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test types helpers", func() {
	Context("Tests namespace unmarshalling", func() {
		namespace := "randomNamespace"
		serviceName := "randomServiceName"

		It("Interface marshals and unmarshals preserving the exact same data", func() {
			svn := MeshService{
				Namespace: namespace,
				Name:      serviceName,
			}

			str := svn.String()
			svn2, err := UnmarshalMeshService(str)

			Expect(err).ToNot(HaveOccurred())
			Expect(*svn2).To(Equal(svn))
		})

		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("/svnc")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("svnc/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("/svnc/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("/")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("")
			Expect(err).To(HaveOccurred())

		})
		It("should fail for incomplete names", func() {
			_, err := UnmarshalMeshService("test")
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Test GetSyntheticService()", func() {
		It("returns MeshService", func() {
			namespace := "-namespace-"
			serviceAccount := "-service-account-"

			sa := K8sServiceAccount{
				Namespace: namespace,
				Name:      serviceAccount,
			}

			actual := sa.GetSyntheticService()

			expected := MeshService{
				Namespace: namespace,
				Name:      fmt.Sprintf("-service-account-.-namespace-.osm.synthetic-%s", SyntheticServiceSuffix),
			}
			Expect(actual).To(Equal(expected))
		})
	})

	Context("Testing ServerName", func() {
		It("should return DNS-1123 of the MeshService struct", func() {
			namespacedService := MeshService{
				Namespace: "namespace-here",
				Name:      "service-name-here",
			}
			actual := namespacedService.ServerName()
			Expect(actual).To(Equal("service-name-here.namespace-here.svc.cluster.local"))
		})
	})
})

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

})

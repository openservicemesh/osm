package catalog

import (
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test catalog functions", func() {
	mc := NewFakeMeshCatalog(testclient.NewSimpleClientset())

	Context("Test GetSMISpec()", func() {
		It("provides the SMI Spec component via Mesh Catalog", func() {
			smiSpec := mc.GetSMISpec()
			Expect(smiSpec).ToNot(BeNil())
		})
	})

	Context("Test getAnnouncementChannels()", func() {
		It("provides the SMI Spec component via Mesh Catalog", func() {
			chans := mc.getAnnouncementChannels()

			// Why exactly 6 channels?
			// Because - 1 for MeshSpec changes + 1 for Cert changes + 1 for Ingress + 1 for a Ticker + 1 Namespace + an endpoint provider
			expectedNumberOfChannels := 6
			Expect(len(chans)).To(Equal(expectedNumberOfChannels))
		})
	})
})

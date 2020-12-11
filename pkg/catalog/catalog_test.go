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

			// Current channels are MeshSpec, CertManager, IngressMonitor, Ticker, Services
			expectedNumberOfChannels := 5
			Expect(len(chans)).To(Equal(expectedNumberOfChannels))
			for _, aChannel := range chans {
				Expect(len(aChannel.announcer)).ToNot(BeZero())
				Expect(aChannel.channel).ToNot(BeNil())
			}
		})
	})
})

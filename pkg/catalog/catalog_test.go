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
})

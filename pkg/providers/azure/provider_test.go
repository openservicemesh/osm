package azure

import (
	"github.com/deislabs/smc/pkg/mesh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Azure Compute Provider", func() {
	Describe("Testing Azure Compute Provider", func() {
		Context("Testing parseAzureID", func() {
			uri := mesh.AzureID("/subscriptions/e3f0/resourceGroups/meshTopology-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz")
			It("returns default value in absence of an env var", func() {
				rg, kind, name, err := parseAzureID(uri)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(rg).To(Equal(resourceGroup("meshTopology-rg")))
				Expect(kind).To(Equal(computeKind("Microsoft.Compute/virtualMachineScaleSets")))
				Expect(name).To(Equal(computeName(("baz"))))
			})
		})
	})
})

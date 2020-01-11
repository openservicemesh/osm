package azure

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v12 "github.com/deislabs/smc/pkg/apis/azureresource/v1"
)

var _ = Describe("Azure Compute Provider", func() {
	Describe("Testing Azure Compute Provider", func() {
		Context("Testing parseAzureID", func() {
			uri := azureID("/subscriptions/e3f0/resourceGroups/meshTopology-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz")
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

var _ = Describe("Azure Compute Provider", func() {
	Describe("Testing Azure Compute Provider", func() {
		Context("Testing parseAzureID", func() {
			It("returns default value in absence of an env var", func() {
				svc := v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"k": "v",
							"a": "b",
						},
					},
				}
				azureResources := []*v12.AzureResource{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"k": "v",
							"a": "b",
							"x": "y",
						},
					},
					Spec: v12.AzureResourceSpec{
						ResourceID: "/one/two/three",
					},
				}}
				azID := matchServiceAzureResource(&svc, azureResources)
				expectedAzureID := []azureID{"/one/two/three"}
				Expect(azID).To(Equal(expectedAzureID))
			})
		})
	})
})

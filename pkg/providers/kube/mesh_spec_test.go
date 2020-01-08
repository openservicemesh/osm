package kube

// func matchServiceAzureResource(svc *v1.Service, azureResourcesList []*smc.AzureResource) mesh.ComputeID {

import (
	v12 "github.com/deislabs/smc/pkg/apis/azureresource/v1"
	"github.com/deislabs/smc/pkg/mesh"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
				computeID := matchServiceAzureResource(&svc, azureResources)
				expectedComputeID := mesh.ComputeID{AzureID: "/one/two/three"}
				Expect(computeID).To(Equal(expectedComputeID))
			})
		})
	})
})

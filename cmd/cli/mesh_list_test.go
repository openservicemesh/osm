package main

import (
	"bytes"
	"context"

	"github.com/onsi/gomega/gstruct"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Running the mesh list command", func() {
	Context("when multiple control planes exist", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
			deployments   *v1.DeploymentList
			listCmd       *meshListCmd
		)

		// helper function that adds deployment to the clientset
		addDeployment := func(depName, meshName, namespace string, isMesh bool) {
			dep := createDeployment(depName, meshName, isMesh)
			_, err := fakeClientSet.AppsV1().Deployments(namespace).Create(context.TODO(), dep, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		// helper function that takes element from slice and returns the name for gomega struct
		// https://onsi.github.io/gomega/#gstruct-testing-complex-data-types
		idSelector := func(element interface{}) string {
			return string(element.(v1.Deployment).ObjectMeta.Name) + "/" + string(element.(v1.Deployment).ObjectMeta.Namespace)
		}

		out = new(bytes.Buffer)
		fakeClientSet = fake.NewSimpleClientset()
		listCmd = &meshListCmd{
			out:       out,
			clientSet: fakeClientSet,
		}

		It("should print only correct meshes", func() {
			addDeployment("osm-controller-1", "testMesh1", "testNs1", true)
			addDeployment("osm-controller-2", "testMesh2", "testNs2", true)
			addDeployment("not-osm-controller", "", "testNs3", false)

			deployments, err = getControllerDeployments(listCmd.clientSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployments.Items).To(gstruct.MatchAllElements(idSelector, gstruct.Elements{
				"osm-controller-1/testNs1": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Namespace": Equal("testNs1"),
						"Labels": gstruct.MatchKeys(gstruct.IgnoreMissing, gstruct.Keys{
							"app":      Equal(constants.OSMControllerName),
							"meshName": Equal("testMesh1"),
						}),
					}),
				}),
				"osm-controller-2/testNs2": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Namespace": Equal("testNs2"),
						"Labels": gstruct.MatchKeys(gstruct.IgnoreMissing, gstruct.Keys{
							"app":      Equal(constants.OSMControllerName),
							"meshName": Equal("testMesh2"),
						}),
					}),
				}),
			}))
		})

	})

	Context("when no control planes exist", func() {
		var (
			out           *bytes.Buffer
			fakeClientSet kubernetes.Interface
			err           error
			listCmd       *meshListCmd
		)

		out = new(bytes.Buffer)
		fakeClientSet = fake.NewSimpleClientset()

		listCmd = &meshListCmd{
			out:       out,
			clientSet: fakeClientSet,
		}

		err = listCmd.run()

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
		It("should print no meshes found message", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(out.String()).To(Equal("No control planes found\n"))
		})
	})
})

func createDeployment(deploymentName, meshName string, isMesh bool) *v1.Deployment {
	labelMap := make(map[string]string)
	if isMesh {
		labelMap["app"] = constants.OSMControllerName
		labelMap["meshName"] = meshName
	} else {
		labelMap["app"] = "non-mesh-app"
	}
	dep := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   deploymentName,
			Labels: labelMap,
		},
	}
	return dep
}

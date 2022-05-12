package main

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	v1 "k8s.io/api/apps/v1"
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
			_, err = addDeployment(fakeClientSet, "osm-controller-1", "testMesh1", "testNs1", "testVersion0.1.2", true)
			Expect(err).NotTo(HaveOccurred())
			_, err = addDeployment(fakeClientSet, "osm-controller-2", "testMesh2", "testNs2", "testVersion0.1.3", true)
			Expect(err).NotTo(HaveOccurred())
			_, err = addDeployment(fakeClientSet, "not-osm-controller", "", "testNs3", "", false)
			Expect(err).NotTo(HaveOccurred())

			deployments, err = getControllerDeployments(listCmd.clientSet)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployments.Items).To(gstruct.MatchAllElements(idSelector, gstruct.Elements{
				"osm-controller-1/testNs1": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Namespace": Equal("testNs1"),
						"Labels": gstruct.MatchKeys(gstruct.IgnoreMissing, gstruct.Keys{
							constants.AppLabel:              Equal(constants.OSMControllerName),
							"meshName":                      Equal("testMesh1"),
							constants.OSMAppVersionLabelKey: Equal("testVersion0.1.2"),
						}),
					}),
				}),
				"osm-controller-2/testNs2": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"ObjectMeta": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Namespace": Equal("testNs2"),
						"Labels": gstruct.MatchKeys(gstruct.IgnoreMissing, gstruct.Keys{
							constants.AppLabel:              Equal(constants.OSMControllerName),
							"meshName":                      Equal("testMesh2"),
							constants.OSMAppVersionLabelKey: Equal("testVersion0.1.3"),
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
			Expect(out.String()).To(Equal("No osm mesh control planes found\n"))
		})
	})
})

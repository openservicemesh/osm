package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test proxy resource setting",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 8,
	},
	func() {
		Context("proxy resources", func() {
			It("tests default resource values and updated resource values for proxies", func() {
				// Install OSM
				Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

				const testNS = "client"
				const clientNoResourceLimits = "client1"
				const clientResLimits = "client2"

				// Create Test NS
				Expect(Td.CreateNs(testNS, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, testNS)).To(Succeed())

				// Confirm there are no resource limits for proxies by default
				meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(len(meshConfig.Spec.Sidecar.Resources.Limits)).To(BeZero())
				Expect(len(meshConfig.Spec.Sidecar.Resources.Requests)).To(BeZero())

				// Create simple app, expect no resources for this proxy
				createSimpleApp(clientNoResourceLimits, testNS)

				// Validate proxy resources for this pod, expect none set
				pod, err := Td.Client.CoreV1().Pods(testNS).Get(context.Background(), clientNoResourceLimits, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				found := false
				for _, cont := range pod.Spec.Containers {
					if cont.Name == "envoy" {
						Expect(len(cont.Resources.Limits)).To(BeZero())
						Expect(len(cont.Resources.Requests)).To(BeZero())
						found = true
					}
				}
				Expect(found).To(BeTrue())

				// Update meshconfig, update proxy resources
				meshConfig.Spec.Sidecar.Resources = corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("128M"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("64M"),
					},
				}
				_, err = Td.ConfigClient.ConfigV1alpha3().MeshConfigs(Td.OsmNamespace).Update(context.TODO(), meshConfig, v1.UpdateOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				// Create a new app
				createSimpleApp(clientResLimits, testNS)

				// Verify this pod has now the correct proxy resource limits and requests set
				pod, err = Td.Client.CoreV1().Pods(testNS).Get(context.Background(), clientResLimits, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				found = false
				for _, cont := range pod.Spec.Containers {
					if cont.Name == "envoy" {
						Expect(cont.Resources.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("2")))
						Expect(cont.Resources.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("128M")))
						Expect(cont.Resources.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("1")))
						Expect(cont.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("64M")))
						found = true
					}
				}
				Expect(found).To(BeTrue())
			})
		})
	})

func createSimpleApp(appName string, ns string) {
	// Get simple pod definitions for the HTTP server
	svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
		SimplePodAppDef{
			PodName:   appName,
			Namespace: ns,
			Command:   []string{"/bin/bash", "-c", "--"},
			Args:      []string{"while true; do sleep 30; done;"},
			Image:     "songrgg/alpine-debug",
			Ports:     []int{80},
			OS:        Td.ClusterOS,
		})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(ns, &svcAccDef)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreatePod(ns, podDef)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreateService(ns, svcDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(ns, 90*time.Second, 1, nil)).To(Succeed())
}

package e2e

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test deployment of Fluent Bit sidecar",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 2,
	},
	func() {
		Context("Fluentbit", func() {
			It("Deploys a Fluent Bit sidecar only when enabled", func() {
				// Install OSM with Fluentbit
				installOpts := Td.GetOSMInstallOpts()
				installOpts.DeployFluentbit = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				pods, err := Td.Client.CoreV1().Pods(Td.OsmNamespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{"app": "osm-controller"}).String(),
				})

				Expect(err).NotTo(HaveOccurred())
				cond := false
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						if strings.Contains(container.Image, "fluent-bit") {
							cond = true
						}
					}
				}
				Expect(cond).To(BeTrue())

				err = Td.DeleteNs(Td.OsmNamespace)
				Expect(err).NotTo(HaveOccurred())
				err = Td.WaitForNamespacesDeleted([]string{Td.OsmNamespace}, 60*time.Second)
				Expect(err).NotTo(HaveOccurred())

				// Install OSM without Fluentbit (default)
				installOpts = Td.GetOSMInstallOpts()
				Expect(Td.InstallOSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 1)).To(Succeed())

				pods, err = Td.Client.CoreV1().Pods(Td.OsmNamespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{"app": "osm-controller"}).String(),
				})

				Expect(err).NotTo(HaveOccurred())
				cond = false
				for _, pod := range pods.Items {
					for _, container := range pod.Spec.Containers {
						if strings.Contains(container.Image, "fluent-bit") {
							cond = true
						}
					}
				}
				Expect(cond).To(BeFalse())
			})
		})
	})

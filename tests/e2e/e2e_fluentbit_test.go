package e2e

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var _ = OSMDescribe("Test deployment of Fluent Bit sidecar",
	OSMDescribeInfo{
		tier:   2,
		bucket: 2,
	},
	func() {
		Context("Fluentbit", func() {
			It("Deploys a Fluent Bit sidecar only when enabled", func() {
				// Install OSM with Fluentbit
				installOpts := td.GetOSMInstallOpts()
				installOpts.deployFluentbit = true
				Expect(td.InstallOSM(installOpts)).To(Succeed())

				pods, err := td.client.CoreV1().Pods(td.osmNamespace).List(context.TODO(), metav1.ListOptions{
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

				err = td.DeleteNs(td.osmNamespace)
				Expect(err).NotTo(HaveOccurred())
				err = td.WaitForNamespacesDeleted([]string{td.osmNamespace}, 60*time.Second)
				Expect(err).NotTo(HaveOccurred())

				// Install OSM without Fluentbit (default)
				installOpts = td.GetOSMInstallOpts()
				Expect(td.InstallOSM(installOpts)).To(Succeed())
				Expect(td.WaitForPodsRunningReady(td.osmNamespace, 60*time.Second, 1)).To(Succeed())

				pods, err = td.client.CoreV1().Pods(td.osmNamespace).List(context.TODO(), metav1.ListOptions{
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

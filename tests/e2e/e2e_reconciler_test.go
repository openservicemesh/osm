package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test OSM Reconciler",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 9,
	},
	func() {
		Context("Enable Reconciler", func() {
			It("Update and delete meshConfig crd", func() {

				// Install OSM with reconciler enabled
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnableReconciler = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				// Get the meshConfig crd
				crd, err := Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				originalSpecServed := crd.Spec.Versions[0].Served

				// update the spec served from true to false
				crd.Spec.Versions[0].Served = false
				_, err = Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), crd, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify that crd remains unchanged
				Eventually(func() (bool, error) {
					updatedCrd, err := Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
					return updatedCrd.Spec.Versions[0].Served, err
				}, 3*time.Second).Should(Equal(originalSpecServed))

				// delete the crd
				err = Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Delete(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify crd exists in the cluster after deletion
				_, err = Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Update and delete mutating webhook configuration", func() {

				// Install OSM with reconciler enabled
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnableReconciler = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				// Get the mutating webhook
				mwhc, err := Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), "osm-webhook-osm", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				originalWebhookServiceName := mwhc.Webhooks[0].ClientConfig.Service.Name

				// update the webhook service name
				mwhc.Webhooks[0].ClientConfig.Service.Name = "random-new-service"
				_, err = Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.Background(), mwhc, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify that mutating webhook remains unchanged
				Eventually(func() (string, error) {
					updatedMwhc, err := Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), "osm-webhook-osm", metav1.GetOptions{})
					return updatedMwhc.Webhooks[0].ClientConfig.Service.Name, err
				}, 3*time.Second).Should(Equal(originalWebhookServiceName))

				// delete the mutating webhook
				err = Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), "osm-webhook-osm", metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify the mutating webhook exists in the cluster after deletion
				Eventually(func() error {
					_, err = Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), "osm-webhook-osm", metav1.GetOptions{})
					return err
				}, 3*time.Second).Should(BeNil())
			})

			It("Update and delete validating webhook configuration", func() {

				// Install OSM with reconciler enabled
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnableReconciler = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				// Get the validating webhook
				vwhc, err := Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "osm-validator-mesh-osm", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				originalWebhookServiceName := vwhc.Webhooks[0].ClientConfig.Service.Name

				// update the webhook service name
				vwhc.Webhooks[0].ClientConfig.Service.Name = "random-new-service"
				_, err = Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.Background(), vwhc, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify that validating webhook remains unchanged
				Eventually(func() (string, error) {
					updatedVwhc, err := Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "osm-validator-mesh-osm", metav1.GetOptions{})
					return updatedVwhc.Webhooks[0].ClientConfig.Service.Name, err
				}, 3*time.Second).Should(Equal(originalWebhookServiceName))

				// delete the validating webhook
				err = Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), "osm-validator-mesh-osm", metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify the validating webhook exists in the cluster after deletion
				Eventually(func() error {
					_, err = Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "osm-validator-mesh-osm", metav1.GetOptions{})
					return err
				}, 3*time.Second).Should(BeNil())
			})
		})
	})

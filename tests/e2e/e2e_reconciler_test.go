package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/reconciler"
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
				}, 30*time.Second).Should(Equal(originalWebhookServiceName))

				// delete the validating webhook
				err = Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.Background(), "osm-validator-mesh-osm", metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				// verify the validating webhook exists in the cluster after deletion
				Eventually(func() error {
					_, err = Td.Client.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), "osm-validator-mesh-osm", metav1.GetOptions{})
					return err
				}, 30*time.Second).Should(BeNil())
			})
		})

		Context("Pre-existing reconciler with CRD conversion webhook change", func() {
			It("Retries until CRD change is detected 3 times, waiting for 5 seconds each time", func() {
				// Install CRDs with a conversion webhook
				crdFiles, err := filepath.Glob("../../cmd/osm-bootstrap/crds/*.yaml")

				Expect(err).NotTo(HaveOccurred())

				scheme := runtime.NewScheme()
				codecs := serializer.NewCodecFactory(scheme)
				decode := codecs.UniversalDeserializer().Decode

				for _, file := range crdFiles {
					yaml, err := os.ReadFile(filepath.Clean(file))
					if err != nil {
						Expect(err).NotTo(HaveOccurred())
					}

					crd := &apiv1.CustomResourceDefinition{}
					_, _, err = decode(yaml, nil, crd)
					if err != nil {
						Expect(err).NotTo(HaveOccurred())
					}

					// imitate older version of OSM who set the reconcile label to a (stringified) bool
					crd.Labels[constants.ReconcileLabel] = strconv.FormatBool(true)

					crd.Spec.Conversion = &apiv1.CustomResourceConversion{
						Strategy: apiv1.WebhookConverter,
						Webhook: &apiv1.WebhookConversion{
							ClientConfig: &apiv1.WebhookClientConfig{
								Service: &apiv1.ServiceReference{
									Namespace: "osm-system",
									Name:      "osm-bootstrap",
									Path:      pointer.StringPtr("/convert"),
									Port:      pointer.Int32Ptr(9443),
								},
							},
							ConversionReviewVersions: []string{"v1alpha1", "v1alpha2", "v1alpha3", "v1beta1", "v1"},
						},
					}

					err = util.CreateApplyAnnotation(crd, unstructured.UnstructuredJSONScheme)
					Expect(err).NotTo(HaveOccurred())

					_, err = Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.Background(), crd, metav1.CreateOptions{})
					if errors.IsAlreadyExists(err) {
						existingCRD, err := Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						existingCRD.Spec.Conversion = crd.Spec.Conversion
						existingCRD.Labels[constants.ReconcileLabel] = strconv.FormatBool(true)
						_, err = Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Update(context.Background(), existingCRD, metav1.UpdateOptions{})
						Expect(err).NotTo(HaveOccurred())
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				}
				// Add existing reconciler to watch the cluster
				_, cancel := context.WithCancel(context.Background())
				defer cancel()
				stop := make(chan struct{})
				Expect(err).NotTo(HaveOccurred())

				// The current reconciler code matches the version to the label value, which is a stringified bool
				err = reconciler.NewReconcilerClient(Td.Client, Td.APIServerClient, "osm", "true", stop, reconciler.CrdInformerKey)
				Expect(err).NotTo(HaveOccurred())

				// Confirm that the CRD conversion webhook is set to WebhookConverter
				crd, err := Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(crd.Spec.Conversion.Strategy).To(Equal(apiv1.WebhookConverter))

				// Install OSM with reconciler enabled
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnableReconciler = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				Eventually(func() int {
					// Make sure the CRD conversion webhook is set to NoneConverter 3 times in a row
					var attempts int
					for i := 0; i < 3; attempts++ {
						if attempts >= 30 { // Gomega doesn't stop the function at the timeout, so prevent an infinite loop
							break
						}
						crd, err := Td.APIServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "meshconfigs.config.openservicemesh.io", metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
						if crd.Spec.Conversion.Strategy == apiv1.NoneConverter {
							i++
						} else {
							i = 0
						}
						time.Sleep(500 * time.Millisecond)
					}
					return attempts
				}, 15*time.Second).Should(BeNumerically("<", 30))

				// Stop the first reconciler (simulate the OSM upgrade completion)
				close(stop)
			})
		})
	})

package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/reconciler"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Reconcile MutatingWebhookConfiguration",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("MutatingWebhookConfigurationReconciler", func() {
			// name of the webhook which the controller watches
			webhookName := "osm-webhook"
			testWebhookServiceNamespace := "test-namespace"
			testWebhookServiceName := "test-service-name"
			testWebhookServicePath := "/path"
			var caBundle []byte

			var (
				stopCh chan struct{}
			)

			BeforeEach(func() {
				stopCh = make(chan struct{})

				mgr, err := ctrl.NewManager(Td.RestConfig, ctrl.Options{
					MetricsBindAddress: "0",
				})
				Expect(err).NotTo(HaveOccurred(), "failed to create manager")

				mockController := gomock.NewController(GinkgoT())
				cfgMock := configurator.NewMockConfigurator(mockController)

				certManager := tresor.NewFakeCertManager(cfgMock)
				cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, testWebhookServiceNamespace))
				validity := 1 * time.Hour
				cert, _ := certManager.IssueCertificate(cn, validity)
				Expect(cert.GetCommonName()).To(Equal(cn))
				Expect((cert.GetCertificateChain())).NotTo(BeNil())
				caBundle = cert.GetCertificateChain()

				// Assume the certificate secret is created on kubernetes, as is required in regular execution
				Expect(Td.CreateNs(testWebhookServiceNamespace, map[string]string{})).To(BeNil())
				_, err = providers.GetCertificateFromSecret(testWebhookServiceNamespace, constants.WebhookCertificateSecretName, cert, Td.Client)
				Expect(err).To(BeNil())

				controller := &reconciler.MutatingWebhookConfigurationReconciler{
					Client:       mgr.GetClient(),
					KubeClient:   Td.Client,
					Scheme:       scheme.Scheme,
					OsmWebhook:   webhookName,
					OsmNamespace: testWebhookServiceNamespace,
				}
				err = controller.SetupWithManager(mgr)
				Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

				go func() {
					defer GinkgoRecover()
					err := mgr.Start(context.TODO())
					Expect(err).NotTo(HaveOccurred(), "failed to start manager")
				}()
			})

			AfterEach(func() {
				close(stopCh)
			})

			It("Should add a CA bundle when OSM webhook is missing one", func() {
				mwhc := getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath)
				_, err := Td.CreateMutatingWebhook(mwhc)
				Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook")

				time.Sleep(time.Second * 1)
				actualMwhc, errMwhc := Td.GetMutatingWebhook(webhookName)
				Expect(errMwhc).NotTo(HaveOccurred())

				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).NotTo(BeNil())
				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).To(Equal(caBundle))
			})

			It("Should not add a CA bundle on a random webhook", func() {
				webhookName = "random-webhook"
				mwhc := getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath)
				_, err := Td.CreateMutatingWebhook(mwhc)
				Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook")

				time.Sleep(time.Second * 1)
				actualMwhc, errMwhc := Td.GetMutatingWebhook(webhookName)
				Expect(errMwhc).NotTo(HaveOccurred())

				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).To(BeNil())
			})
		})
		AfterEach(func() {
			// Cleanup
			err := Td.Client.AdmissionregistrationV1().MutatingWebhookConfigurations().DeleteCollection(context.Background(),
				metav1.DeleteOptions{}, metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(Td.GetTestNamespaceSelectorMap()).String(),
				})
			Expect(err).To(BeNil())
		})
	})

func getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath string) *admissionregv1.MutatingWebhookConfiguration {
	return &admissionregv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   webhookName,
			Labels: Td.GetTestNamespaceSelectorMap(),
		},
		Webhooks: []admissionregv1.MutatingWebhook{
			{
				Name: injector.MutatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					Service: &admissionregv1.ServiceReference{
						Namespace: testWebhookServiceNamespace,
						Name:      testWebhookServiceName,
						Path:      &testWebhookServicePath,
					},
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"some-key": "some-value",
					},
				},
			},
		},
	}
}

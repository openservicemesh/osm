package e2e

import (
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	reconciler "github.com/openservicemesh/osm/pkg/reconciler/mutatingwebhook"
)

var _ = OSMDescribe("Reconcile MutatingWebhookConfiguration",
	OSMDescribeInfo{
		tier:   1,
		bucket: 2,
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

				mgr, err := ctrl.NewManager(td.restConfig, ctrl.Options{
					MetricsBindAddress: "0",
				})
				Expect(err).NotTo(HaveOccurred(), "failed to create manager")

				mockController := gomock.NewController(GinkgoT())
				cfgMock := configurator.NewMockConfigurator(mockController)
				cache := make(map[certificate.CommonName]certificate.Certificater)
				certManager := tresor.NewFakeCertManager(&cache, cfgMock)
				cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, testWebhookServiceNamespace))
				validity := 1 * time.Hour
				cert, _ := certManager.IssueCertificate(cn, validity)
				Expect(cert.GetCommonName()).To(Equal(cn))
				Expect((cert.GetCertificateChain())).NotTo(BeNil())
				caBundle = cert.GetCertificateChain()

				controller := &reconciler.MutatingWebhookConfigrationReconciler{
					Client:       mgr.GetClient(),
					Scheme:       scheme.Scheme,
					OsmWebhook:   webhookName,
					OsmNamespace: testWebhookServiceNamespace,
					CertManager:  certManager,
				}
				err = controller.SetupWithManager(mgr)
				Expect(err).NotTo(HaveOccurred(), "failed to setup controller")

				go func() {
					err := mgr.Start(stopCh)
					Expect(err).NotTo(HaveOccurred(), "failed to start manager")
				}()
			})

			AfterEach(func() {
				close(stopCh)
			})

			It("Should add a CA bundle when OSM webhook is missing one", func() {
				mwhc := getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath)
				_, err := td.CreateMutatingWebhook(mwhc)
				Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook")

				time.Sleep(time.Second * 1)
				actualMwhc, errMwhc := td.GetMutatingWebhook(webhookName)
				Expect(errMwhc).NotTo(HaveOccurred())

				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).NotTo(BeNil())
				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).To(Equal(caBundle))
			})

			It("Should not add a CA bundle on a random webhook", func() {
				webhookName = "random-webhook"
				mwhc := getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath)
				_, err := td.CreateMutatingWebhook(mwhc)
				Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook")

				time.Sleep(time.Second * 1)
				actualMwhc, errMwhc := td.GetMutatingWebhook(webhookName)
				Expect(errMwhc).NotTo(HaveOccurred())

				Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).To(BeNil())
			})
		})
	})

func getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath string) *v1beta1.MutatingWebhookConfiguration {
	return &v1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []v1beta1.MutatingWebhook{
			{
				Name: injector.MutatingWebhookName,
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: testWebhookServiceNamespace,
						Name:      testWebhookServiceName,
						Path:      &testWebhookServicePath,
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"some-key": "some-value",
					},
				},
			},
		},
	}
}

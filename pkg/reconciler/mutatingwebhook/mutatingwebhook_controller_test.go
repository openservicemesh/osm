package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("MutatingWehookConfigurationReconciler", func() {
	Context("Testing reconcile for MutatingWebhookConfiguration", func() {

		webhookName := "osm-webhook"
		testWebhookServiceNamespace := "test-namespace"
		testWebhookServiceName := "test-service-name"
		testWebhookServicePath := "/path"

		var (
			ctx    context.Context
			stopCh chan struct{}
			ns     *corev1.Namespace
		)

		BeforeEach(func() {
			stopCh = make(chan struct{})
			ctx = context.TODO()

			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: testWebhookServiceNamespace},
			}

			err := k8sClient.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{})
			Expect(err).NotTo(HaveOccurred(), "failed to create manager")

			mockController := gomock.NewController(GinkgoT())
			cfg := configurator.NewMockConfigurator(mockController)
			cache := make(map[certificate.CommonName]certificate.Certificater)
			certManager := tresor.NewFakeCertManager(&cache, cfg)
			cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, testWebhookServiceNamespace))
			validity := 1 * time.Hour
			cert, err := certManager.IssueCertificate(cn, validity)
			Expect(cert.GetCommonName()).To(Equal(cn))
			Expect((cert.GetCertificateChain())).NotTo(BeNil())

			controller := &MutatingWebhookConfigrationReconciler{
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

			err := k8sClient.Delete(ctx, ns)
			Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
		})

		It("Should add the CA bundle", func() {
			mwhc := getTestMWHC(webhookName, ns.Name, testWebhookServiceName, testWebhookServicePath)

			err := k8sClient.Create(ctx, mwhc)
			Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook resource")

			actualMwhc := &v1beta1.MutatingWebhookConfiguration{}

			Eventually(
				getResourceFunc(ctx, client.ObjectKey{Name: webhookName, Namespace: mwhc.Namespace}, actualMwhc),
				time.Second*5, 10*time.Millisecond).Should(BeNil())

			fmt.Printf("Checking mutating webhook")
			Expect(actualMwhc.Webhooks[0].Name).To(Equal("osm-inject.k8s.io"))
			Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).NotTo(BeNil())
			Expect(actualMwhc.Webhooks[0].ClientConfig.CABundle).To(Equal([]byte{}))

		})

		/*It("Should not add the CA bundle", func() {
			mwhc := getTestMWHC("random-webhook", testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath)

			err := k8sClient.Create(ctx, mwhc)
			Expect(err).NotTo(HaveOccurred(), "failed to create test mutating webhook resource")

			actualMWHC := &v1beta1.MutatingWebhookConfiguration{}

			Eventually(
				getResourceFunc(ctx, client.ObjectKey{Name: webhookName, Namespace: mwhc.Namespace}, actualMWHC),
				time.Second*5, 10*time.Millisecond).Should(BeNil())

			Expect(actualMWHC.Webhooks[0].ClientConfig.CABundle).To(BeNil())
		})*/
	})
})

func getTestMWHC(webhookName, testWebhookServiceNamespace, testWebhookServiceName, testWebhookServicePath string) *v1beta1.MutatingWebhookConfiguration {
	return &v1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
		Webhooks: []v1beta1.MutatingWebhook{
			{
				Name: "osm-inject.k8s.io",
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

package injector

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

var _ = Describe("Test MutatingWebhookConfiguration patch", func() {
	Context("find and patches webhook", func() {
		//cert := tresor.Certificate{}
		cert := mockCertificate{}
		meshName := "--meshName--"
		webhookName := "--webhookName--"
		//TODO:seed a test webhook
		testWebhookServiceNamespace := "test-namespace"
		testWebhookServiceName := "test-service-name"
		testWebhookServicePath := "/path"
		kubeClient := fake.NewSimpleClientset(&admissionv1beta1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookName,
			},
			Webhooks: []admissionv1beta1.MutatingWebhook{
				{
					Name: osmWebhookName,
					ClientConfig: admissionv1beta1.WebhookClientConfig{
						Service: &admissionv1beta1.ServiceReference{
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
		})

		It("checks if the hook exists", func() {
			err := hookExists(kubeClient, webhookName)
			Expect(err).ToNot(HaveOccurred())
		})

		It("checks if a non existent hook exists", func() {
			err := hookExists(kubeClient, webhookName+"blah")
			Expect(err).To(HaveOccurred())
		})

		It("patches a webhook", func() {
			err := patchMutatingWebhookConfiguration(cert, meshName, webhookName, kubeClient)
			Expect(err).ToNot(HaveOccurred())

		})

		It("ensures webhook is configured correctly", func() {
			webhooks, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(webhooks.Items)).To(Equal(1))

			webhook := webhooks.Items[0]
			Expect(len(webhook.Webhooks)).To(Equal(1))
			Expect(webhook.Webhooks[0].NamespaceSelector.MatchLabels["some-key"]).To(Equal("some-value"))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Namespace).To(Equal(testWebhookServiceNamespace))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Name).To(Equal(testWebhookServiceName))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Path).To(Equal(&testWebhookServicePath))
			Expect(webhook.Webhooks[0].ClientConfig.CABundle).To(Equal([]byte("chain")))
			Expect(len(webhook.Webhooks[0].Rules)).To(Equal(1))
			rule := webhook.Webhooks[0].Rules[0]
			Expect(len(rule.Operations)).To(Equal(1))
			Expect(rule.Operations[0]).To(Equal(admissionv1beta1.Create))
			Expect(rule.Rule.APIGroups).To(Equal([]string{""}))
			Expect(rule.Rule.APIVersions).To(Equal([]string{"v1"}))
			Expect(rule.Rule.Resources).To(Equal([]string{"pods"}))
		})
	})
})

type mockCertificate struct{}

func (mc mockCertificate) GetCommonName() certificate.CommonName { return "" }
func (mc mockCertificate) GetCertificateChain() []byte           { return []byte("chain") }
func (mc mockCertificate) GetPrivateKey() []byte                 { return []byte("key") }
func (mc mockCertificate) GetIssuingCA() []byte                  { return []byte("ca") }
func (mc mockCertificate) GetExpiration() time.Time              { return time.Now() }

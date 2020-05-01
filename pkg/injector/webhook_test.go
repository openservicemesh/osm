package injector

import (
	"context"

	"github.com/open-service-mesh/osm/pkg/namespace"

	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	// v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test MutatingWebhookConfiguration creation", func() {
	Context("create webhook", func() {

		cert := tresor.Certificate{}
		osmID := "--osmID--"
		osmNamespace := "--namespace--"
		webhookName := "--webhookName--"
		kubeClient := testclient.NewSimpleClientset()

		It("creates a webhook", func() {
			err := createMutatingWebhookConfiguration(cert, osmID, osmNamespace, webhookName, kubeClient)
			Expect(err).ToNot(HaveOccurred())

		})

		It("checks if the hook exists", func() {
			actual, err := hookExists(kubeClient, webhookName)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(BeTrue())
		})

		It("checks if a non existent hook exists", func() {
			actual, err := hookExists(kubeClient, webhookName+"blah")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(BeFalse())
		})

		It("only one webhook is created", func() {
			webhooks, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(context.TODO(), v1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(webhooks.Items)).To(Equal(1))

			webhook := webhooks.Items[0]
			Expect(len(webhook.Webhooks)).To(Equal(1))
			Expect(webhook.Webhooks[0].NamespaceSelector.MatchLabels[namespace.MonitorLabel]).To(Equal(osmID))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Namespace).To(Equal(osmNamespace))
			Expect(webhook.Webhooks[0].ClientConfig.Service.Name).To(Equal(osmWebhookServiceName))
			Expect(*(webhook.Webhooks[0].ClientConfig.Service.Path)).To(Equal(osmWebhookMutatePath))
		})
	})
})

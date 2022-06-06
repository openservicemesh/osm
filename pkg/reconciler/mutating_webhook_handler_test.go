package reconciler

import (
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

var testWebhookServicePath = "/path"

func TestMutatingWebhookEventHandlerUpdateFunc(t *testing.T) {
	testCases := []struct {
		name         string
		originalMwhc admissionv1.MutatingWebhookConfiguration
		updatedMwhc  admissionv1.MutatingWebhookConfiguration
		mwhcUpdated  bool
	}{
		{
			name: "webhook name and namespace selector changed",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "new-test-service-name-",
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
			},
			mwhcUpdated: true,
		},
		{
			name: "mutating webhook new label added",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
						"some":                           "label",
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			mwhcUpdated: false,
		},
		{
			name: "mutataing webhook name changed",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--updatedWebhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			mwhcUpdated: true,
		},
	}

	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.ReconciliationTotal)
	defer metricsstore.DefaultMetricsStore.Stop(metricsstore.DefaultMetricsStore.ReconciliationTotal)
	expectedMetric := 0

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)

			kubeClient := testclient.NewSimpleClientset(&tc.originalMwhc)

			c := client{
				kubeClient:      kubeClient,
				meshName:        meshName,
				osmVersion:      osmVersion,
				apiServerClient: nil,
				informers:       informerCollection{},
			}
			// Invoke update handler
			handlers := c.mutatingWebhookEventHandler()
			handlers.UpdateFunc(&tc.originalMwhc, &tc.updatedMwhc)

			if tc.mwhcUpdated {
				a.Equal(&tc.originalMwhc, &tc.updatedMwhc)
			} else {
				a.NotEqual(&tc.originalMwhc, &tc.updatedMwhc)
			}

			crd, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), tc.originalMwhc.Name, metav1.GetOptions{})
			a.Nil(err)

			if tc.mwhcUpdated {
				a.Equal(crd, &tc.updatedMwhc)
			} else {
				a.Equal(crd, &tc.originalMwhc)
			}

			if tc.mwhcUpdated {
				expectedMetric++
			}
			if expectedMetric > 0 {
				a.True(metricsstore.DefaultMetricsStore.Contains(`osm_reconciliation_total{kind="MutatingWebhookConfiguration"} ` + strconv.Itoa(expectedMetric) + "\n"))
			}
		})
	}
}

func TestMutatingWebhookEventHandlerDeleteFunc(t *testing.T) {
	originalMwhc := admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "--webhookName--",
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.ReconcileLabel:         strconv.FormatBool(true),
				constants.AppLabel:               constants.OSMInjectorName,
				constants.OSMAppVersionLabelKey:  osmVersion,
				constants.OSMAppInstanceLabelKey: meshName,
			},
		},
		Webhooks: []admissionv1.MutatingWebhook{
			{
				Name: injector.MutatingWebhookName,
				ClientConfig: v1.WebhookClientConfig{
					Service: &v1.ServiceReference{
						Namespace: "test-namespace",
						Name:      "test-service-name",
						Path:      &testWebhookServicePath,
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
			},
		},
	}

	a := tassert.New(t)
	kubeClient := testclient.NewSimpleClientset()

	c := client{
		kubeClient:      kubeClient,
		meshName:        meshName,
		osmVersion:      osmVersion,
		apiServerClient: nil,
		informers:       informerCollection{},
	}
	// Invoke delete handler
	handlers := c.mutatingWebhookEventHandler()
	handlers.DeleteFunc(&originalMwhc)

	// verify mwhc exists after deletion
	mwhc, err := c.kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), originalMwhc.Name, metav1.GetOptions{})
	a.Nil(err)
	a.Equal(mwhc, &originalMwhc)
}

func TestIsMutatingWebhookUpdated(t *testing.T) {
	testCases := []struct {
		name         string
		originalMwhc admissionv1.MutatingWebhookConfiguration
		updatedMwhc  admissionv1.MutatingWebhookConfiguration
		mwhcUpdated  bool
	}{
		{
			name: "webhook and namespace selector changed",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "new-test-service-name-",
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
			},
			mwhcUpdated: true,
		},
		{
			name: "mutating webhook new label added",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
						"some":                           "label",
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			mwhcUpdated: false,
		},
		{
			name: "mutataing webhook name changed",
			originalMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--webhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			updatedMwhc: admissionv1.MutatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "--updatedWebhookName--",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:         strconv.FormatBool(true),
						constants.AppLabel:               constants.OSMInjectorName,
						constants.OSMAppVersionLabelKey:  osmVersion,
						constants.OSMAppInstanceLabelKey: meshName,
					},
				},
				Webhooks: []admissionv1.MutatingWebhook{
					{
						Name: injector.MutatingWebhookName,
						ClientConfig: v1.WebhookClientConfig{
							Service: &v1.ServiceReference{
								Namespace: "test-namespace",
								Name:      "test-service-name",
								Path:      &testWebhookServicePath,
							},
						},
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
								constants.OSMAppInstanceLabelKey: meshName,
							},
						},
					},
				},
			},
			mwhcUpdated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			c := client{
				osmVersion: osmVersion,
			}
			result := c.isMutatingWebhookUpdated(&tc.originalMwhc, &tc.updatedMwhc)
			assert.Equal(result, tc.mwhcUpdated)
		})
	}
}

package validator

import (
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	ingressRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"policy.openservicemesh.io"},
			APIVersions: []string{"v1alpha1"},
			Resources:   []string{"ingressbackends", "egresses"},
		},
	}

	trafficTargetRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"access.smi-spec.io"},
			APIVersions: []string{"v1alpha3"},
			Resources:   []string{"traffictargets"},
		},
	}

	configRule = admissionregv1.RuleWithOperations{
		Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
		Rule: admissionregv1.Rule{
			APIGroups:   []string{"config.openservicemesh.io"},
			APIVersions: []string{"v1alpha2"},
			Resources:   []string{"meshrootcertificates"},
		},
	}
)

func TestCreateValidatingWebhook(t *testing.T) {
	webhookPath := validationAPIPath
	webhookPort := int32(constants.ValidatorWebhookPort)
	osmVersion := "test-version"
	webhookName := "--webhookName--"
	meshName := "test-mesh"
	osmNamespace := "test-namespace"
	enableReconciler := true

	testCases := []struct {
		name                      string
		validateTrafficTarget     bool
		expectedRules             []admissionregv1.RuleWithOperations
		expectedControlPlaneRules []admissionregv1.RuleWithOperations
	}{
		{
			name:                      "with smi validation enabled",
			validateTrafficTarget:     true,
			expectedRules:             []admissionregv1.RuleWithOperations{ingressRule, trafficTargetRule},
			expectedControlPlaneRules: []admissionregv1.RuleWithOperations{configRule},
		},
		{
			name:                      "with smi validation disabled",
			validateTrafficTarget:     false,
			expectedRules:             []admissionregv1.RuleWithOperations{ingressRule},
			expectedControlPlaneRules: []admissionregv1.RuleWithOperations{configRule},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			cert := &certificate.Certificate{}

			kubeClient := fake.NewSimpleClientset()

			err := createOrUpdateValidatingWebhook(kubeClient, cert, webhookName, meshName, osmNamespace, osmVersion, tc.validateTrafficTarget, enableReconciler)
			assert.Nil(err)
			webhooks, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Len(webhooks.Items, 1)

			wh := webhooks.Items[0]
			assert.Len(wh.Webhooks, 2)
			assert.Equal(wh.ObjectMeta.Name, webhookName)
			assert.EqualValues(wh.ObjectMeta.Labels, map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.OSMAppInstanceLabelKey: meshName,
				constants.OSMAppVersionLabelKey:  osmVersion,
				constants.AppLabel:               constants.OSMControllerName,
				constants.ReconcileLabel:         strconv.FormatBool(true),
			})

			for _, webhook := range wh.Webhooks {
				assert.Equal(webhook.ClientConfig.Service.Namespace, osmNamespace)
				assert.Equal(webhook.ClientConfig.Service.Name, ValidatorWebhookSvc)
				assert.Equal(webhook.ClientConfig.Service.Path, &webhookPath)
				assert.Equal(webhook.ClientConfig.Service.Port, &webhookPort)
				assert.Equal(webhook.AdmissionReviewVersions, []string{"v1"})

				if webhook.Name == ValidatingWebhookName {
					assert.Equal(webhook.NamespaceSelector.MatchLabels[constants.OSMKubeResourceMonitorAnnotation], meshName)
					assert.EqualValues(webhook.NamespaceSelector.MatchExpressions, []metav1.LabelSelectorRequirement{
						{
							Key:      constants.IgnoreLabel,
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
						{
							Key:      "name",
							Operator: metav1.LabelSelectorOpNotIn,
							Values:   []string{osmNamespace},
						},
						{
							Key:      "control-plane",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					})
					assert.ElementsMatch(webhook.Rules, tc.expectedRules)
				} else if webhook.Name == ControlPlaneValidatingWebhookName {
					assert.EqualValues(webhook.NamespaceSelector.MatchExpressions, []metav1.LabelSelectorRequirement{
						{
							Key:      "name",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{osmNamespace},
						},
					})
					assert.ElementsMatch(webhook.Rules, tc.expectedControlPlaneRules)
				} else {
					assert.Fail("unknown webhook %s in validating webhook configuration", webhook.Name)
				}
			}
		})
	}
}

package validator

import (
	"context"
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestCreateValidatingWebhook(t *testing.T) {
	assert := tassert.New(t)
	webhookName := "--webhookName--"
	meshName := "test-mesh"
	osmNamespace := "test-namespace"
	osmVersion := "test-version"
	webhookPath := validationAPIPath
	webhookPort := int32(constants.ValidatorWebhookPort)

	mockCtrl := gomock.NewController(t)
	cert := certificate.NewMockCertificater(mockCtrl)
	cert.EXPECT().GetCertificateChain()

	kubeClient := fake.NewSimpleClientset()
	err := createValidatingWebhook(kubeClient, cert, webhookName, meshName, osmNamespace, osmVersion)
	assert.Nil(err)

	webhooks, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.TODO(), metav1.ListOptions{})
	assert.Nil(err)
	assert.Len(webhooks.Items, 1)

	wh := webhooks.Items[0]
	assert.Len(wh.Webhooks, 1)
	assert.Equal(wh.ObjectMeta.Name, webhookName)
	assert.EqualValues(wh.ObjectMeta.Labels, map[string]string{
		constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
		constants.OSMAppInstanceLabelKey: meshName,
		constants.OSMAppVersionLabelKey:  osmVersion,
		"app":                            constants.OSMControllerName,
		constants.ReconcileLabel:         strconv.FormatBool(true),
	})

	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Namespace, osmNamespace)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Name, ValidatorWebhookSvc)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Path, &webhookPath)
	assert.Equal(wh.Webhooks[0].ClientConfig.Service.Port, &webhookPort)

	assert.Equal(wh.Webhooks[0].NamespaceSelector.MatchLabels[constants.OSMKubeResourceMonitorAnnotation], meshName)
	assert.EqualValues(wh.Webhooks[0].NamespaceSelector.MatchExpressions, []metav1.LabelSelectorRequirement{
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
	assert.ElementsMatch(wh.Webhooks[0].Rules, []admissionregv1.RuleWithOperations{
		{
			Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
			Rule: admissionregv1.Rule{
				APIGroups:   []string{"policy.openservicemesh.io"},
				APIVersions: []string{"v1alpha1"},
				Resources:   []string{"ingressbackends", "egresses"},
			},
		},
	})
	assert.Equal(wh.Webhooks[0].AdmissionReviewVersions, []string{"v1"})
}

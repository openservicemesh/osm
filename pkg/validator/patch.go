package validator

import (
	"context"
	"encoding/json"
	"strconv"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
)

const (
	// ValidatingWebhookName is the name of the validating webhook.
	ValidatingWebhookName = "osm-validator.k8s.io"

	// ValidatorWebhookSvc is the name of the validator service.
	ValidatorWebhookSvc = "osm-validator"
)

// getPartialValidatingWebhookConfiguration returns only the portion of the ValidatingWebhookConfiguration that needs
// to be updated.
func getPartialValidatingWebhookConfiguration(name string, cert certificate.Certificater) admissionregv1.ValidatingWebhookConfiguration {
	return admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Webhooks: []admissionregv1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					CABundle: cert.GetCertificateChain(),
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
}

// updateValidatingWebhookCABundle updates the existing ValidatingWebhookConfiguration with the CA this OSM instance runs with.
// It is necessary to perform this patch because the original ValidatingWebhookConfig YAML does not contain the root certificate.
func updateValidatingWebhookCABundle(webhookConfigName string, certificater certificate.Certificater, kubeClient kubernetes.Interface) error {
	vwc := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations()

	patchJSON, err := json.Marshal(getPartialValidatingWebhookConfiguration(webhookConfigName, certificater))
	if err != nil {
		return err
	}

	if _, err = vwc.Patch(context.Background(), webhookConfigName, types.StrategicMergePatchType, patchJSON, metav1.PatchOptions{}); err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingValidatingWebhookCABundle)).
			Msgf("Error updating CA Bundle for ValidatingWebhookConfiguration %s", webhookConfigName)
		return err
	}

	log.Info().Msgf("Finished updating CA Bundle for ValidatingWebhookConfiguration %s", webhookConfigName)
	return nil
}

func createValidatingWebhook(clientSet kubernetes.Interface, cert certificate.Certificater, webhookName, meshName, osmNamespace, osmVersion string) error {
	webhookPath := validationAPIPath
	webhookPort := int32(constants.ValidatorWebhookPort)
	failuerPolicy := admissionregv1.Fail
	matchPolict := admissionregv1.Exact

	vwhc := admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.OSMAppInstanceLabelKey: meshName,
				constants.OSMAppVersionLabelKey:  osmVersion,
				"app":                            constants.OSMControllerName,
				constants.ReconcileLabel:         strconv.FormatBool(true)}},
		Webhooks: []admissionregv1.ValidatingWebhook{
			{
				Name: ValidatingWebhookName,
				ClientConfig: admissionregv1.WebhookClientConfig{
					Service: &admissionregv1.ServiceReference{
						Namespace: osmNamespace,
						Name:      ValidatorWebhookSvc,
						Path:      &webhookPath,
						Port:      &webhookPort,
					},
					CABundle: cert.GetCertificateChain()},
				FailurePolicy: &failuerPolicy,
				MatchPolicy:   &matchPolict,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.OSMKubeResourceMonitorAnnotation: meshName,
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
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
					},
				},
				Rules: []admissionregv1.RuleWithOperations{
					{
						Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
						Rule: admissionregv1.Rule{
							APIGroups:   []string{"policy.openservicemesh.io"},
							APIVersions: []string{"v1alpha1"},
							Resources:   []string{"ingressbackends", "egresses"},
						},
					},
				},
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"}}},
	}

	if _, err := clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), &vwhc, metav1.CreateOptions{}); err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingValidatingWebhook)).
			Msgf("Error creating ValidatingWebhookConfiguration %s", webhookName)
		return err
	}

	log.Info().Msgf("Finished creating ValidatingWebhookConfiguration %s", webhookName)
	return nil
}

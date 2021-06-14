package validator

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
)

const (
	// ValidatingWebhookName is the name of the validating webhook.
	ValidatingWebhookName = "osm-validate.k8s.io"
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
		log.Error().Err(err).Msgf("Error updating CA Bundle for ValidatingWebhookConfiguration %s", webhookConfigName)
		return err
	}

	log.Info().Msgf("Finished updating CA Bundle for ValidatingWebhookConfiguration %s", webhookConfigName)
	return nil
}

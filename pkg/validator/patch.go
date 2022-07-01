package validator

import (
	"context"
	"strconv"

	admissionregv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func createOrUpdateValidatingWebhook(clientSet kubernetes.Interface, cert *certificate.Certificate, webhookName, meshName, osmNamespace, osmVersion string, validateTrafficTarget bool, enableReconciler bool) error {
	webhookPath := validationAPIPath
	webhookPort := int32(constants.ValidatorWebhookPort)
	failurePolicy := admissionregv1.Fail
	matchPolicy := admissionregv1.Exact

	rules := []admissionregv1.RuleWithOperations{
		{
			Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
			Rule: admissionregv1.Rule{
				APIGroups:   []string{"policy.openservicemesh.io"},
				APIVersions: []string{"v1alpha1"},
				Resources:   []string{"ingressbackends", "egresses"},
			},
		},
	}

	if validateTrafficTarget {
		rules = append(rules, admissionregv1.RuleWithOperations{
			Operations: []admissionregv1.OperationType{admissionregv1.Create, admissionregv1.Update},
			Rule: admissionregv1.Rule{
				APIGroups:   []string{"access.smi-spec.io"},
				APIVersions: []string{"v1alpha3"},
				Resources:   []string{"traffictargets"},
			},
		})
	}

	vwhcLabels := map[string]string{
		constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
		constants.OSMAppInstanceLabelKey: meshName,
		constants.OSMAppVersionLabelKey:  osmVersion,
		constants.AppLabel:               constants.OSMControllerName,
	}

	if enableReconciler {
		vwhcLabels[constants.ReconcileLabel] = strconv.FormatBool(true)
	}

	vwhc := admissionregv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   webhookName,
			Labels: vwhcLabels,
		},
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
					CABundle: cert.GetTrustedCAs()},
				FailurePolicy: &failurePolicy,
				MatchPolicy:   &matchPolicy,
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
				Rules: rules,
				SideEffects: func() *admissionregv1.SideEffectClass {
					sideEffect := admissionregv1.SideEffectClassNoneOnDryRun
					return &sideEffect
				}(),
				AdmissionReviewVersions: []string{"v1"}}},
	}

	if _, err := clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), &vwhc, metav1.CreateOptions{}); err != nil {
		// Webhook already exists, update the webhook in this scenario
		if apierrors.IsAlreadyExists(err) {
			existing, err := clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), vwhc.Name, metav1.GetOptions{})
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMutatingWebhook)).
					Msgf("Error getting ValidatingWebhookConfiguration %s", webhookName)
				return err
			}

			vwhc.ObjectMeta = existing.ObjectMeta // copy the object meta which includes resource version, required for updates.
			if _, err = clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.Background(), &vwhc, metav1.UpdateOptions{}); err != nil {
				// There might be conflicts when multiple controllers try to update the same resource
				// One of the controllers will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
				if !apierrors.IsConflict(err) {
					log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingMutatingWebhook)).
						Msgf("Error updating ValidatingWebhookConfiguration %s with error %v", webhookName, err)
					return err
				}
			}
		} else {
			// Webhook doesn't exist and could not be created, an error is logged and returned
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingValidatingWebhook)).
				Msgf("Error creating ValidatingWebhookConfiguration %s", webhookName)
			return err
		}
	}

	log.Info().Msgf("Finished creating ValidatingWebhookConfiguration %s", webhookName)
	return nil
}

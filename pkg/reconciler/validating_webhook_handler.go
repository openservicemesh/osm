package reconciler

import (
	"context"
	reflect "reflect"
	"strings"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// validatingWebhookEventHandler creates validating webhook events handlers.
func (c client) validatingWebhookEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldVwhc := oldObj.(*admissionv1.ValidatingWebhookConfiguration)
			newVwhc := newObj.(*admissionv1.ValidatingWebhookConfiguration)
			log.Debug().Msgf("validating webhook update event for %s", newVwhc.Name)
			if !c.isValidatingWebhookUpdated(oldVwhc, newVwhc) {
				return
			}

			c.reconcileValidatingWebhook(oldVwhc, newVwhc)
		},

		DeleteFunc: func(obj interface{}) {
			vwhc := obj.(*admissionv1.ValidatingWebhookConfiguration)
			c.addValidatingWebhook(vwhc)
			log.Debug().Msgf("validating webhook delete event for %s", vwhc.Name)
		},
	}
}

func (c client) reconcileValidatingWebhook(oldVwhc, newVwhc *admissionv1.ValidatingWebhookConfiguration) {
	newVwhc.Webhooks = oldVwhc.Webhooks
	newVwhc.ObjectMeta.Name = oldVwhc.ObjectMeta.Name
	newVwhc.ObjectMeta.Labels = oldVwhc.ObjectMeta.Labels
	if _, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.Background(), newVwhc, metav1.UpdateOptions{}); err != nil {
		// There might be conflicts when multiple controllers try to update the same resource
		// One of the controllers will successfully update the resource, hence conflicts shoud be ignored and not treated as an error
		if !apierrors.IsConflict(err) {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingUpdatedValidatingWebhook)).
				Msgf("Error updating validating webhook: %s with error %v", newVwhc.Name, err)
		}
	}
	log.Debug().Msgf("Successfully reconciled validating webhook %s", newVwhc.Name)
}

func (c client) addValidatingWebhook(oldVwhc *admissionv1.ValidatingWebhookConfiguration) {
	oldVwhc.ResourceVersion = ""
	if _, err := c.kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.Background(), oldVwhc, metav1.CreateOptions{}); err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrReconcilingDeletedValidatingWebhook)).
			Msgf("Error adding back deleted validating webhook: %s", oldVwhc.Name)
	}
	log.Debug().Msgf("Successfully added back validating webhook %s", oldVwhc.Name)
}

func (c *client) isValidatingWebhookUpdated(oldVwhc, newVwhc *admissionv1.ValidatingWebhookConfiguration) bool {
	webhookEqual := reflect.DeepEqual(oldVwhc.Webhooks, newVwhc.Webhooks)
	vwhcNameChanged := strings.Compare(oldVwhc.ObjectMeta.Name, newVwhc.ObjectMeta.Name) != 0
	vwhcLabelsChanged := isLabelModified(constants.AppLabel, constants.OSMControllerName, newVwhc.ObjectMeta.Labels) ||
		isLabelModified(constants.OSMAppVersionLabelKey, c.osmVersion, newVwhc.ObjectMeta.Labels)
	vwhcUpdated := !webhookEqual || vwhcNameChanged || vwhcLabelsChanged
	return vwhcUpdated
}

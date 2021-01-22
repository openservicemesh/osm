package reconciler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("reconciler")

// MutatingWebhookConfigurationReconciler reconciles a MutatingWebhookConfiguration object
type MutatingWebhookConfigurationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	OsmWebhook   string
	OsmNamespace string
	CertManager  certificate.Manager
}

// Reconcile is the reconciliation method for OSM MutatingWebhookConfiguration.
func (r *MutatingWebhookConfigurationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	// reconcile only for OSM mutatingWebhookConfiguration
	if req.Name == r.OsmWebhook {
		ctx := context.Background()
		instance := &v1beta1.MutatingWebhookConfiguration{}

		if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
			log.Error().Err(err).Msgf("Error reading object %s ", req.NamespacedName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		var shouldUpdate bool
		if r.OsmWebhook == instance.Name {
			//check if CA bundle exists on webhook
			for idx, webhook := range instance.Webhooks {
				// CA bundle missing for webhook, update webhook to include the latest CA bundle
				if webhook.Name == injector.MutatingWebhookName && webhook.ClientConfig.CABundle == nil {
					log.Trace().Msgf("CA bundle missing for webhook : %s ", req.Name)
					shouldUpdate = true
					cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, r.OsmNamespace))
					cert, err := r.CertManager.GetCertificate(cn)
					if err != nil {
						return ctrl.Result{}, errors.Errorf("Error updating mutating webhook, unable to get certificate for the mutating webhook %s: %s", req.Name, err)
					}
					instance.Webhooks[idx].ClientConfig.CABundle = cert.GetCertificateChain()
				}
			}
		}

		if !shouldUpdate {
			log.Trace().Msgf("Mutatingwebhookconfiguration %s already compliant", req.Name)
			return ctrl.Result{}, nil
		}

		if err := r.Update(ctx, instance); err != nil {
			log.Error().Err(err).Msgf("Error updating MutatingWebhookConfiguration %s", req.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Debug().Msgf("Successfully updated CA Bundle for MutatingWebhookConfiguration %s ", req.Name)

		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager links the reconciler to the manager.
func (r *MutatingWebhookConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.MutatingWebhookConfiguration{}).
		Complete(r)
}

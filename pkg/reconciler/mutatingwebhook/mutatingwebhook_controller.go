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
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("mutatingwebhook-reconciler")

// MutatingWebhookConfigrationReconciler reconciles a MutatingWebhookConfiguration object
type MutatingWebhookConfigrationReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	OsmWebhook   string
	OsmNamespace string
	CertManager  certificate.Manager
}

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations/status,verbs=get;update;patch

// Reconcile is the reconciliation method for OSM MutatingWebhookConfiguration.
func (r *MutatingWebhookConfigrationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	instance := &v1beta1.MutatingWebhookConfiguration{}

	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		log.Error().Err(err).Msgf("failure reading object %s ", req.NamespacedName.String())
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// reconcile only for OSM mutatingWebhookConfiguration
	var shouldUpdate bool
	if r.OsmWebhook == instance.Name {
		log.Trace().Msgf("TEST Trying to update mutating webhook configuration for : %s", req.Name)
		//check if CA bundle exists on webhook
		for idx, webhook := range instance.Webhooks {
			// CA bundle missing for webhook, update webhook to include the latest CA bundle
			if webhook.Name == "osm-inject.k8s.io" && webhook.ClientConfig.CABundle == nil {
				log.Trace().Msgf("TEST CA bundle missing for webhook : %s ", req.Name)
				shouldUpdate = true
				cn := certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMControllerName, r.OsmNamespace))
				cert, err := r.CertManager.GetCertificate(cn)
				if err != nil {
					return ctrl.Result{}, errors.Errorf("TEST Unable to update mutating webhook, Error getting certificate for the mutating webhook: %+v", err)
				}
				instance.Webhooks[idx].ClientConfig.CABundle = cert.GetCertificateChain()
			}
		}
	}

	if !shouldUpdate {
		log.Trace().Msgf("TEST mutating webhook configuration already compliant %s", req.Name)
		return ctrl.Result{}, nil
	}

	if err := r.Update(ctx, instance); err != nil {
		log.Error().Err(err).Msgf("failure reading object %s", req.NamespacedName.String())
		return ctrl.Result{Requeue: false}, nil
	}

	log.Trace().Msgf("TEST Successfully updated mutating webhook configuration CA bundle for : %s ", req.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager links the reconciler to the manager.
func (r *MutatingWebhookConfigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.MutatingWebhookConfiguration{}).
		Complete(r)
}

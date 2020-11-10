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
	// reconcile only for OSM mutatingWebhookConfiguration
	if req.Name == r.OsmWebhook {
		ctx := context.Background()
		instance := &v1beta1.MutatingWebhookConfiguration{}

		if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
			log.Error().Err(err).Msgf("failure reading object %s ", req.NamespacedName.String())
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
						return ctrl.Result{}, errors.Errorf("Error updating mutating webhook, unable to get certificate for the mutating webhook %s: %+s", req.Name, err)
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
			log.Error().Err(err).Msgf("Error updating mutatingwebhookconfiguration %s: %s", req.Name, err)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Trace().Msgf("Successfully updated mutatingwebhookconfiguration CA bundle for : %s ", req.Name)

		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager links the reconciler to the manager.
func (r *MutatingWebhookConfigrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.MutatingWebhookConfiguration{}).
		Complete(r)
}

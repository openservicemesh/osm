package main

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/reconciler"
)

// createReconciler sets up k8s controller manager to reconcile osm-injector's mutatingwehbookconfiguration
func createReconciler(kubeClient *kubernetes.Clientset) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0", /* disables controller manager metrics serving */
		Namespace:          osmNamespace,
	})
	if err != nil {
		return errors.Wrap(err, "Error creating controller-runtime Manager for MutatingWebhookConfiguration's reconciler")
	}

	// Add a reconciler for osm-injector's mutatingwehbookconfiguration
	if err = (&reconciler.MutatingWebhookConfigurationReconciler{
		Client:       mgr.GetClient(),
		KubeClient:   kubeClient,
		Scheme:       mgr.GetScheme(),
		OsmWebhook:   webhookConfigName,
		OsmNamespace: osmNamespace,
	}).SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "Error creating controller to reconcile MutatingWebhookConfiguration")
	}

	go func() {
		// mgr.Start() below will block until stopped
		// See: https://github.com/kubernetes-sigs/controller-runtime/blob/release-0.6/pkg/manager/internal.go#L507-L514
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrStartingReconcileManager)).
				Msg("Error starting controller-runtime manager for MutatingWebhookConfigurartion's reconciler")
		}
	}()

	return nil
}

package main

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

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
		log.Error().Err(err).Msg("Error creating controller manager")
		return err
	}

	// Add a reconciler for osm-injector's mutatingwehbookconfiguration
	if err = (&reconciler.MutatingWebhookConfigurationReconciler{
		Client:       mgr.GetClient(),
		KubeClient:   kubeClient,
		Scheme:       mgr.GetScheme(),
		OsmWebhook:   fmt.Sprintf("osm-webhook-%s", meshName),
		OsmNamespace: osmNamespace,
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Msg("Error creating controller to reconcile MutatingWebhookConfiguration")
		return err
	}

	go func() {
		// mgr.Start() below will block until stopped
		// See: https://github.com/kubernetes-sigs/controller-runtime/blob/release-0.6/pkg/manager/internal.go#L507-L514
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			log.Error().Err(err).Msg("Error setting up signal handler for reconciler")
		}
	}()

	return nil
}

package crdconversion

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// webhookHealthPath is the HTTP path at which the health of the conversion webhook can be queried
	webhookHealthPath = "/healthz"

	// healthPort is the port on which the '/healthz` requests are served
	healthPort = 9095

	// paths to convert CRD's
	trafficAccessConverterPath         = "/convert/trafficaccess"
	httpRouteGroupConverterPath        = "/convert/httproutegroup"
	meshConfigConverterPath            = "/convert/meshconfig"
	multiclusterServiceConverterPath   = "/convert/multiclusterservice"
	egressPolicyConverterPath          = "/convert/egresspolicy"
	trafficSplitConverterPath          = "/convert/trafficsplit"
	tcpRoutesConverterPath             = "/convert/tcproutes"
	ingressBackendsPolicyConverterPath = "/convert/ingressbackendspolicy"
)

var crdConversionWebhookConfiguration = map[string]string{
	"traffictargets.access.smi-spec.io":              trafficAccessConverterPath,
	"httproutegroups.specs.smi-spec.io":              httpRouteGroupConverterPath,
	"meshconfigs.config.openservicemesh.io":          meshConfigConverterPath,
	"multiclusterservices.config.openservicemesh.io": multiclusterServiceConverterPath,
	"egresses.policy.openservicemesh.io":             egressPolicyConverterPath,
	"trafficsplits.split.smi-spec.io":                trafficSplitConverterPath,
	"tcproutes.specs.smi-spec.io":                    tcpRoutesConverterPath,
	"ingressbackends.policy.openservicemesh.io":      ingressBackendsPolicyConverterPath,
}

var conversionReviewVersions = []string{"v1beta1", "v1"}

// NewConversionWebhook starts a new web server handling requests from the CRD's
func NewConversionWebhook(config Config, kubeClient kubernetes.Interface, crdClient apiclient.ApiextensionsV1Interface, certManager certificate.Manager, osmNamespace string, enableReconciler bool, stop <-chan struct{}) error {
	// This is a certificate issued for the crd-converter webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned with the ConversionWebhook on the CRD's
	crdConversionWebhookHandlerCert, err := certManager.IssueCertificate(
		certificate.CommonName(fmt.Sprintf("%s.%s.svc", constants.OSMBootstrapName, osmNamespace)),
		constants.XDSCertificateValidityPeriod)
	if err != nil {
		return errors.Errorf("Error issuing certificate for the crd-converter: %+v", err)
	}

	// The following function ensures to atomically create or get the certificate from Kubernetes
	// secret API store. Multiple instances should end up with the same crdConversionwebhookHandlerCert after this function executed.
	crdConversionWebhookHandlerCert, err = providers.GetCertificateFromSecret(osmNamespace, constants.CrdConverterCertificateSecretName, crdConversionWebhookHandlerCert, kubeClient)
	if err != nil {
		return errors.Errorf("Error fetching crd-converter certificate from k8s secret: %s", err)
	}

	crdWh := crdConversionWebhook{
		config: config,
		cert:   crdConversionWebhookHandlerCert,
	}

	// Start the ConversionWebhook web server
	go crdWh.run(stop)

	if err = patchCrds(crdConversionWebhookHandlerCert, crdClient, osmNamespace, enableReconciler); err != nil {
		return errors.Errorf("Error patching crds with conversion webhook %v", err)
	}

	return nil
}

func (crdWh *crdConversionWebhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	webhookMux := http.NewServeMux()
	webhookMux.HandleFunc(meshConfigConverterPath, serveMeshConfigConversion)
	webhookMux.HandleFunc(trafficAccessConverterPath, serveTrafficAccessConversion)
	webhookMux.HandleFunc(httpRouteGroupConverterPath, serveHTTPRouteGroupConversion)
	webhookMux.HandleFunc(multiclusterServiceConverterPath, serveMultiClusterServiceConversion)
	webhookMux.HandleFunc(egressPolicyConverterPath, serveEgressPolicyConversion)
	webhookMux.HandleFunc(trafficSplitConverterPath, serveTrafficSplitConversion)
	webhookMux.HandleFunc(tcpRoutesConverterPath, serveTCPRouteConversion)
	webhookMux.HandleFunc(ingressBackendsPolicyConverterPath, serveIngressBackendsPolicyConversion)

	webhookServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", crdWh.config.ListenPort),
		Handler: webhookMux,
	}

	log.Info().Msgf("Starting conversion webhook server on port: %v", crdWh.config.ListenPort)
	go func() {
		// Generate a key pair from your pem-encoded cert and key ([]byte).
		cert, err := tls.X509KeyPair(crdWh.cert.GetCertificateChain(), crdWh.cert.GetPrivateKey())
		if err != nil {
			log.Error().Err(err).Msg("Error parsing crd-converter webhook certificate")
			return
		}

		// #nosec G402
		webhookServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := webhookServer.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msg("crd-converter webhook HTTP server failed to start")
			return
		}
	}()

	healthMux := http.NewServeMux()
	healthMux.HandleFunc(webhookHealthPath, healthHandler)

	healthServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", healthPort),
		Handler: healthMux,
	}

	go func() {
		if err := healthServer.ListenAndServe(); err != nil {
			log.Error().Err(err).Msg("crd-converter health server failed to start")
			return
		}
	}()

	// Wait on exit signals
	<-stop

	// Stop the servers
	if err := webhookServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down crd-conversion webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down crd-conversion webhook HTTP server")
	}

	if err := healthServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down crd-conversion health server")
	} else {
		log.Info().Msg("Done shutting down crd-conversion health server")
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msg("Error writing bytes for crd-conversion webhook health check handler")
	}
}

func patchCrds(cert certificate.Certificater, crdClient apiclient.ApiextensionsV1Interface, osmNamespace string, enableReconciler bool) error {
	for crdName, crdConversionPath := range crdConversionWebhookConfiguration {
		if err := updateCrdConfiguration(cert, crdClient, osmNamespace, crdName, crdConversionPath, enableReconciler); err != nil {
			log.Error().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crdName)
			return err
		}
	}
	return nil
}

// updateCrdConfiguration updates the Conversion section of the CRD and adds a reconcile label if OSM's reconciler is enabled.
func updateCrdConfiguration(cert certificate.Certificater, crdClient apiclient.ApiextensionsV1Interface, osmNamespace, crdName, crdConversionPath string, enableReconciler bool) error {
	crd, err := crdClient.CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	crd.Spec.Conversion = &apiv1.CustomResourceConversion{
		Strategy: apiv1.WebhookConverter,
		Webhook: &apiv1.WebhookConversion{
			ClientConfig: &apiv1.WebhookClientConfig{
				Service: &apiv1.ServiceReference{
					Namespace: osmNamespace,
					Name:      constants.OSMBootstrapName,
					Path:      &crdConversionPath,
				},
				CABundle: cert.GetCertificateChain(),
			},
			ConversionReviewVersions: conversionReviewVersions,
		},
	}

	if enableReconciler {
		existingLabels := crd.Labels
		if existingLabels == nil {
			existingLabels = map[string]string{}
		}
		existingLabels[constants.ReconcileLabel] = strconv.FormatBool(true)
		crd.Labels = existingLabels
	}

	if _, err = crdClient.CustomResourceDefinitions().Update(context.Background(), crd, metav1.UpdateOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crdName)
		return err
	}

	log.Info().Msgf("successfully updated conversion webhook configuration for crd : %s", crdName)
	return nil
}

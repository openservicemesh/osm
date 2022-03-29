package crdconversion

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/metricsstore"
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
	retryPolicyConverterPath           = "/convert/retrypolicy"
)

// apiKindToPath maps the resource API kind to the HTTP path at which
// the webhook server peforms the conversion
// *Note: only add API kinds for which conversion is necessary so that
// the webhook is not invoked otherwise.
var apiKindToPath = map[string]string{
	"meshconfigs.config.openservicemesh.io": meshConfigConverterPath,
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
	handlers := map[string]http.HandlerFunc{
		meshConfigConverterPath:            serveMeshConfigConversion,
		trafficAccessConverterPath:         serveTrafficAccessConversion,
		httpRouteGroupConverterPath:        serveHTTPRouteGroupConversion,
		multiclusterServiceConverterPath:   serveMultiClusterServiceConversion,
		egressPolicyConverterPath:          serveEgressPolicyConversion,
		trafficSplitConverterPath:          serveTrafficSplitConversion,
		tcpRoutesConverterPath:             serveTCPRouteConversion,
		ingressBackendsPolicyConverterPath: serveIngressBackendsPolicyConversion,
		retryPolicyConverterPath:           serveRetryPolicyConversion,
	}
	for endpoint, h := range handlers {
		webhookMux.Handle(endpoint, metricsstore.AddHTTPMetrics(h))
	}

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
			MinVersion:   tls.VersionTLS13,
		}

		if err := webhookServer.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msg("crd-converter webhook HTTP server failed to start")
			return
		}
	}()

	healthMux := http.NewServeMux()
	healthMux.Handle(webhookHealthPath, metricsstore.AddHTTPMetrics(http.HandlerFunc(healthHandler)))

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

func patchCrds(cert *certificate.Certificate, crdClient apiclient.ApiextensionsV1Interface, osmNamespace string, enableReconciler bool) error {
	for crdName, crdConversionPath := range apiKindToPath {
		if err := updateCrdConfiguration(cert, crdClient, osmNamespace, crdName, crdConversionPath); err != nil {
			log.Error().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crdName)
			return err
		}
	}
	return nil
}

// updateCrdConfiguration updates the Conversion section of the CRD and adds a reconcile label if OSM's reconciler is enabled.
func updateCrdConfiguration(cert *certificate.Certificate, crdClient apiclient.ApiextensionsV1Interface, osmNamespace, crdName, crdConversionPath string) error {
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
					Port:      pointer.Int32(constants.CRDConversionWebhookPort),
					Path:      &crdConversionPath,
				},
				CABundle: cert.GetIssuingCA(),
			},
			ConversionReviewVersions: conversionReviewVersions,
		},
	}

	if _, err = crdClient.CustomResourceDefinitions().Update(context.Background(), crd, metav1.UpdateOptions{}); err != nil {
		log.Error().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crdName)
		return err
	}

	log.Info().Msgf("successfully updated conversion webhook configuration for crd : %s", crdName)
	return nil
}

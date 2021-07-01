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

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers"
	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	// webhookHealthPath is the HTTP path at which the health of the conversion webhook can be queried
	webhookHealthPath = "/healthz"

	// crdConverterServiceName is the name of the OSM crd converter webhook service
	crdConverterServiceName = "osm-crd-converter"
)

var crdConversionWebhookConfiguration = map[string]string{
	"traffictargets.access.smi-spec.io":              "/trafficaccessconversion",
	"httproutegroups.specs.smi-spec.io":              "/httproutegroupconversion",
	"meshconfigs.config.openservicemesh.io":          "/meshconfigconversion",
	"multiclusterservices.config.openservicemesh.io": "/multiclusterserviceconversion",
	"egresses.policy.openservicemesh.io":             "/egresspolicyconversion",
	"trafficsplits.split.smi-spec.io":                "/trafficsplitconversion",
	"tcproutes.specs.smi-spec.io":                    "/tcproutesconversion",
}

var conversionReviewVersions = []string{"v1beta1", "v1"}

// NewConversionWebhook starts a new web server handling requests from the CRD's
func NewConversionWebhook(config Config, kubeClient kubernetes.Interface, crdClient apiclient.ApiextensionsV1Interface, certManager certificate.Manager, osmNamespace string, stop <-chan struct{}) error {
	// This is a certificate issued for the crd-converter webhook handler
	// This cert does not have to be related to the Envoy certs, but it does have to match
	// the cert provisioned with the ConversionWebhook on the CRD's
	crdConversionWebhookHandlerCert, err := certManager.IssueCertificate(
		certificate.CommonName(fmt.Sprintf("%s.%s.svc", crdConverterServiceName, osmNamespace)),
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

	if err = patchCrdsWithConversionWehook(crdConversionWebhookHandlerCert, crdClient, osmNamespace); err != nil {
		return errors.Errorf("Error patching crds with conversion webhook %v", err)
	}

	return nil
}

func (crdWh *crdConversionWebhook) run(stop <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()

	mux.HandleFunc(webhookHealthPath, healthHandler)

	// TODO (snchh): add handler and logic for conversion stratergy of each CRD in OSM

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", crdWh.config.ListenPort),
		Handler: mux,
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
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Error().Err(err).Msg("crd-converter webhook HTTP server failed to start")
			return
		}
	}()

	// Wait on exit signals
	<-stop

	// Stop the server
	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Error shutting down crd-conversion webhook HTTP server")
	} else {
		log.Info().Msg("Done shutting down crd-conversion webhook HTTP server")
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Health OK")); err != nil {
		log.Error().Err(err).Msg("Error writing bytes for crd-conversion webhook health check handler")
	}
}

func patchCrdsWithConversionWehook(cert certificate.Certificater, crdClient apiclient.ApiextensionsV1Interface, osmNamespace string) error {
	for crdName, crdConversionPath := range crdConversionWebhookConfiguration {
		if err := updateCrdConversionWebhookConfiguration(cert, crdClient, osmNamespace, crdName, crdConversionPath); err != nil {
			log.Error().Err(err).Msgf("Error updating conversion webhook configuration for crd : %s", crdName)
			return err
		}
	}
	return nil
}

// updateCrdConversionWebhookConfiguration updates the Conversion section of the CRD that needs to be updated.
func updateCrdConversionWebhookConfiguration(cert certificate.Certificater, crdClient apiclient.ApiextensionsV1Interface, osmNamespace, crdName, crdConversionPath string) error {
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
					Name:      crdConverterServiceName,
					Path:      &crdConversionPath,
				},
				CABundle: cert.GetCertificateChain(),
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

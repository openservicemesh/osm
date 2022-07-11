package crdconversion

import (
	"context"
	"net/http"

	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/webhook"
)

const (
	// paths to convert CRD's
	trafficAccessConverterPath         = "/convert/trafficaccess"
	httpRouteGroupConverterPath        = "/convert/httproutegroup"
	meshConfigConverterPath            = "/convert/meshconfig"
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
func NewConversionWebhook(ctx context.Context, kubeClient kubernetes.Interface, crdClient apiclient.ApiextensionsV1Interface, certManager *certificate.Manager, osmNamespace string, enableReconciler bool) error {
	srv, err := webhook.NewServer(constants.OSMBootstrapName, osmNamespace, constants.CRDConversionWebhookPort, certManager, map[string]http.HandlerFunc{
		meshConfigConverterPath:            serveMeshConfigConversion,
		trafficAccessConverterPath:         serveTrafficAccessConversion,
		httpRouteGroupConverterPath:        serveHTTPRouteGroupConversion,
		egressPolicyConverterPath:          serveEgressPolicyConversion,
		trafficSplitConverterPath:          serveTrafficSplitConversion,
		tcpRoutesConverterPath:             serveTCPRouteConversion,
		ingressBackendsPolicyConverterPath: serveIngressBackendsPolicyConversion,
		retryPolicyConverterPath:           serveRetryPolicyConversion,
	}, func(cert *certificate.Certificate) error {
		return patchCrds(cert, crdClient, osmNamespace, enableReconciler)
	})
	if err != nil {
		return err
	}

	go srv.Run(ctx)
	return nil
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
				CABundle: cert.GetTrustedCAs(),
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

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
)

type certificateManagerKind string

// These are the supported certificate issuers.
const (
	// Tresor is an internal package, which leverages Kubernetes secrets and signs certs on the OSM pod
	tresorKind certificateManagerKind = "tresor"

	// Azure Key Vault integration; uses AKV for certificate storage only; certs are signed on the OSM pod
	keyVaultKind = "keyvault"

	// Hashi Vault integration; OSM is pointed to an external Vault; signing of certs happens on Vault
	vaultKind = "vault"

	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

// Functions we can call to create a Certificate Manager for each kind of supported certificate issuer
var certManagers = map[certificateManagerKind]func(kubeClient kubernetes.Interface, enableDebugServer bool) (certificate.Manager, debugger.CertificateManagerDebugger, error){
	tresorKind:   getTresorCertificateManager,
	keyVaultKind: getAzureKeyVaultCertManager,
	vaultKind:    getHashiVaultCertManager,
}

// Get a list of the supported certificate issuers
func getPossibleCertManagers() []string {
	var possible []string
	for kind := range certManagers {
		possible = append(possible, string(kind))
	}
	return possible
}

func getTresorCertificateManager(kubeClient kubernetes.Interface, enableDebug bool) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	var err error
	var rootCert certificate.Certificater

	// A non-empty caBundleSecretName indicates to the certificate issuer to
	// load the CA from the given k8s secret within the namespace where OSM is install.d
	// An empty string or nil value would not load or save/load CA.
	if caBundleSecretName != "" {
		rootCert = getCertFromKubernetes(kubeClient, osmNamespace, caBundleSecretName)
	}

	if rootCert == nil {
		rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

		if err != nil {
			return nil, nil, errors.Errorf("Failed to create new Certificate Authority with cert issuer %s", *certManagerKind)
		}

		if rootCert == nil {
			return nil, nil, errors.Errorf("Invalid root certificate created by cert issuer %s", *certManagerKind)
		}

		if rootCert.GetPrivateKey() == nil {
			return nil, nil, errors.Errorf("Root cert does not have a private key")
		}
	}

	certManager, err := tresor.NewCertManager(rootCert, getServiceCertValidityPeriod(), rootCertOrganization)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to instantiate Azure Key Vault as a Certificate Manager")
	}

	return certManager, certManager, nil
}

func getCertFromKubernetes(kubeClient kubernetes.Interface, namespace, secretName string) certificate.Certificater {
	secrets, err := kubeClient.CoreV1().Secrets(namespace).List(context.Background(), v1.ListOptions{})
	if err != nil {
		log.Error().Err(err).Msgf("Error listing secrets in namespace %q", namespace)
	}
	found := false
	for _, secret := range secrets.Items {
		if secret.Name == secretName {
			found = true
			break
		}
	}

	if !found {
		return nil
	}

	rootCertSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, v1.GetOptions{})
	if err != nil {
		log.Warn().Msgf("Error retrieving root certificate rootCertSecret %q from namespace %q; Will create a new one", secretName, osmNamespace)
		return nil
	}

	pemCert, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		log.Error().Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretCAKey)
		return nil
	}

	pemKey, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]
	if !ok {
		log.Error().Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil
	}

	expirationBytes, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAExpiration]
	if !ok {
		log.Error().Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretCAExpiration)
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(string(expirationBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Error decoding base64 encoded CA expiration %q from Kubernetes rootCertSecret %q from namespace %q", expirationBytes, secretName, osmNamespace)
	}

	expirationString := string(decoded)
	expiration, err := time.Parse(constants.TimeDateLayout, expirationString)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing CA expiration %q from Kubernetes rootCertSecret %q from namespace %q", expirationString, secretName, osmNamespace)
	}

	rootCert, err := tresor.NewCertificateFromPEM(pemCert, pemKey, expiration)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", *certManagerKind)
	}
	return rootCert
}

func getAzureKeyVaultCertManager(_ kubernetes.Interface, enableDebug bool) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	// TODO(draychev): implement: https://github.com/openservicemesh/osm/issues/577
	log.Fatal().Msg("Azure Key Vault certificate manager is not implemented")
	return nil, nil, nil
}

func getHashiVaultCertManager(_ kubernetes.Interface, enableDebug bool) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[*vaultProtocol]; !ok {
		return nil, nil, errors.Errorf("Value %s is not a valid Hashi Vault protocol", *vaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", *vaultProtocol, *vaultHost, *vaultPort)
	vaultCertManager, err := vault.NewCertManager(vaultAddr, *vaultToken, getServiceCertValidityPeriod(), *vaultRole)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Hashicorp Vault as a Certificate Manager: %+v", err)
	}

	return vaultCertManager, vaultCertManager, nil
}

func getServiceCertValidityPeriod() time.Duration {
	return time.Duration(serviceCertValidityMinutes) * time.Minute
}

func encodeExpiration(expiration time.Time) []byte {
	// Serialize CA expiration
	expirationString := expiration.Format(constants.TimeDateLayout)
	b64Encoded := make([]byte, base64.StdEncoding.EncodedLen(len(expirationString)))
	base64.StdEncoding.Encode(b64Encoded, []byte(expirationString))
	return b64Encoded
}

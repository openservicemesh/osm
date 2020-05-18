package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/vault"
	"github.com/open-service-mesh/osm/pkg/constants"
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

	// Name of the Kubernetes secret where we store the Root certificate for the service mesh
	rootCertSecretName = "root-cert"

	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

// Functions we can call to create a Certificate Manager for each kind of supported certificate issuer
var certManagers = map[certificateManagerKind]func(kubeConfig *rest.Config) certificate.Manager{
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

func getNewRootCertFromTresor(kubeClient kubernetes.Interface, namespace, rootCertSecretName string, saveRootPrivateKeyInKubernetes bool) certificate.Certificater {
	rootCert, err := tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootExpiration, rootCertCountry, rootCertLocality, rootCertOrganization)

	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", *certManagerKind)
	}

	if rootCert == nil {
		log.Fatal().Msgf("Invalid root certificate created by cert issuer %s", *certManagerKind)
	}

	if rootCert.GetPrivateKey() == nil {
		log.Fatal().Err(err).Msg("Root cert does not have a private key")
	}

	if saveRootPrivateKeyInKubernetes {
		if err := saveSecretToKubernetes(kubeClient, rootCert, namespace, rootCertSecretName, rootCert.GetPrivateKey()); err != nil {
			log.Error().Err(err).Msgf("Error exporting CA bundle into Kubernetes secret with name %s", rootCertSecretName)
		}
	}

	return rootCert
}

func getTresorCertificateManager(kubeConfig *rest.Config) certificate.Manager {
	kubeClient := kubernetes.NewForConfigOrDie(kubeConfig)

	var err error
	var rootCert certificate.Certificater
	if keepRootPrivateKeyInKubernetes {
		rootCert = getCertFromKubernetes(kubeClient, osmNamespace, rootCertSecretName)
		if rootCert == nil {
			rootCert = getNewRootCertFromTresor(kubeClient, osmNamespace, rootCertSecretName, keepRootPrivateKeyInKubernetes)
		}
	} else {
		rootCert = getNewRootCertFromTresor(kubeClient, osmNamespace, rootCertSecretName, keepRootPrivateKeyInKubernetes)
	}

	certManager, err := tresor.NewCertManager(rootCert, getServiceCertValidityPeriod())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to instantiate Azure Key Vault as a Certificate Manager")
	}

	return certManager
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

func getAzureKeyVaultCertManager(_ *rest.Config) certificate.Manager {
	// TODO(draychev): implement: https://github.com/open-service-mesh/osm/issues/577
	log.Fatal().Msg("Azure Key Vault certificate manager is not implemented")
	return nil
}

func getHashiVaultCertManager(_ *rest.Config) certificate.Manager {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[*vaultProtocol]; !ok {
		log.Fatal().Msgf("Value %s is not a valid Hashi Vault protocol", *vaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", *vaultProtocol, *vaultHost, *vaultPort)
	vaultCertManager, err := vault.NewCertManager(vaultAddr, *vaultToken, getServiceCertValidityPeriod())
	if err != nil {
		log.Fatal().Err(err).Msg("Error instantiating Hashicorp Vault as a Certificate Manager")
	}

	_, err = vaultCertManager.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootExpiration)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
	}

	return vaultCertManager
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

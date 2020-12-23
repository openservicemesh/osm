package main

import (
	"context"
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/certmanager"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/debugger"
)

// These are the supported certificate issuers.
const (
	// Tresor is an internal package, which leverages Kubernetes secrets and signs certs on the OSM pod
	tresorKind string = "tresor"

	// Hashi Vault integration; OSM is pointed to an external Vault; signing of certs happens on Vault
	vaultKind = "vault"

	// cert-manager integration; certificates are requested using cert-manager
	// CertificateRequest resources, signed by the configured issuer.
	certmanagerKind = "cert-manager"

	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

var validCertificateManagerOptions = []string{tresorKind, vaultKind, certmanagerKind}

func getTresorOSMCertificateManager(kubeClient kubernetes.Interface, cfg configurator.Configurator) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	var err error
	var rootCert certificate.Certificater

	// A non-empty caBundleSecretName indicates to the certificate issuer to
	// load the CA from the given k8s secret within the namespace where OSM is install.d
	// An empty string or nil value would not load or save/load CA.
	if caBundleSecretName != "" {
		rootCert, err = getCertFromKubernetes(kubeClient, osmNamespace, caBundleSecretName)
		if err != nil {
			log.Error().Err(err).Msgf("Error retrieving root certificate from secret %s/%s", osmNamespace, caBundleSecretName)
			return nil, nil, err
		}
	}

	if rootCert == nil {
		rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

		if err != nil {
			return nil, nil, errors.Errorf("Failed to create new Certificate Authority with cert issuer %s", *osmCertificateManagerKind)
		}

		if rootCert == nil {
			return nil, nil, errors.Errorf("Invalid root certificate created by cert issuer %s", *osmCertificateManagerKind)
		}

		if rootCert.GetPrivateKey() == nil {
			return nil, nil, errors.Errorf("Root cert does not have a private key")
		}
	}

	certManager, err := tresor.NewCertManager(rootCert, rootCertOrganization, cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Failed to instantiate Tresor as a Certificate Manager")
	}

	return certManager, certManager, nil
}

// getCertFromKubernetes returns a Certificater type corresponding to the root certificate.
// The function returns an error only if a secret is found with invalid data.
func getCertFromKubernetes(kubeClient kubernetes.Interface, namespace, secretName string) (certificate.Certificater, error) {
	rootCertSecret, err := kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		// It is okay for this secret to be missing, in which case a new CA will be created along with a k8s secret
		log.Debug().Msgf("Could not retrieve root certificate secret %q from namespace %q", secretName, osmNamespace)
		return nil, nil
	}

	pemCert, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretCAKey)
		return nil, errInvalidCertSecret
	}

	pemKey, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil, errInvalidCertSecret
	}

	expirationBytes, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAExpiration]
	if !ok {
		log.Error().Err(errInvalidCertSecret).Msgf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, secretName, constants.KubernetesOpaqueSecretCAExpiration)
		return nil, errInvalidCertSecret
	}

	expiration, err := time.Parse(constants.TimeDateLayout, string(expirationBytes))
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing CA expiration %q from Kubernetes rootCertSecret %q from namespace %q", string(expirationBytes), secretName, osmNamespace)
		return nil, err
	}

	rootCert, err := tresor.NewCertificateFromPEM(pemCert, pemKey, expiration)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create new Certificate Authority with cert issuer %s", *osmCertificateManagerKind)
		return nil, err
	}

	return rootCert, nil
}

func getHashiVaultOSMCertificateManager(cfg configurator.Configurator) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	if _, ok := map[string]interface{}{"http": nil, "https": nil}[*vaultProtocol]; !ok {
		return nil, nil, errors.Errorf("Value %s is not a valid Hashi Vault protocol", *vaultProtocol)
	}

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", *vaultProtocol, *vaultHost, *vaultPort)
	vaultCertManager, err := vault.NewCertManager(vaultAddr, *vaultToken, *vaultRole, cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Hashicorp Vault as a Certificate Manager: %+v", err)
	}

	return vaultCertManager, vaultCertManager, nil
}

func getCertManagerOSMCertificateManager(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator) (certificate.Manager, debugger.CertificateManagerDebugger, error) {
	rootCertSecret, err := kubeClient.CoreV1().Secrets(osmNamespace).Get(context.TODO(), caBundleSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get cert-manager CA secret %s/%s: %s", osmNamespace, caBundleSecretName, err)
	}

	pemCert, ok := rootCertSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		return nil, nil, fmt.Errorf("Opaque k8s secret %s/%s does not have required field %q", osmNamespace, caBundleSecretName, constants.KubernetesOpaqueSecretCAKey)
	}

	rootCert, err := certmanager.NewRootCertificateFromPEM(pemCert)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode cert-manager CA certificate from secret %s/%s: %s", osmNamespace, caBundleSecretName, err)
	}

	client, err := cmversionedclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to build cert-manager client set: %s", err)
	}

	certmanagerCertManager, err := certmanager.NewCertManager(rootCert, client, osmNamespace, cmmeta.ObjectReference{
		Name:  *certmanagerIssuerName,
		Kind:  *certmanagerIssuerKind,
		Group: *certmanagerIssuerGroup,
	}, cfg)
	if err != nil {
		return nil, nil, errors.Errorf("Error instantiating Jetstack cert-manager as a Certificate Manager: %+v", err)
	}

	return certmanagerCertManager, certmanagerCertManager, nil
}

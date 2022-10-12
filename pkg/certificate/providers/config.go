package providers

import (
	"context"
	"errors"
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	k8storage "github.com/openservicemesh/osm/pkg/certificate/castorage/k8s"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/certmanager"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

var getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
	cert, err := i.IssueCertificate(certificate.NewCertOptionsWithFullName("init-cert", 1*time.Second))
	if err != nil {
		return nil, err
	}

	return cert.GetIssuingCA(), nil
}

// NewCertificateManager returns a new certificate manager with a MRC compat client.
// TODO(4713): Remove and use NewCertificateManagerFromMRC
func NewCertificateManager(ctx context.Context, kubeClient kubernetes.Interface, kubeConfig *rest.Config,
	providerNamespace string, option Options, computeClient compute.Interface, checkInterval time.Duration, trustDomain string) (*certificate.Manager, error) {
	if err := option.Validate(); err != nil {
		return nil, err
	}

	mrcClient := &MRCCompatClient{
		MRCProviderGenerator: MRCProviderGenerator{
			kubeClient:      kubeClient,
			kubeConfig:      kubeConfig,
			KeyBitSize:      utils.GetCertKeyBitSize(computeClient.GetMeshConfig()),
			caExtractorFunc: getCA,
		},
		mrc: &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "legacy-compat",
				Namespace: providerNamespace,
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				Provider:    option.AsProviderSpec(),
				TrustDomain: trustDomain,
				Intent:      constants.MRCIntentActive,
			},
		},
	}
	// TODO(#4745): Remove after deprecating the osm.vault.token option.
	if vaultOption, ok := option.(VaultOptions); ok {
		mrcClient.MRCProviderGenerator.DefaultVaultToken = vaultOption.VaultToken
	}

	return certificate.NewManager(
		ctx,
		mrcClient,
		func() time.Duration { return utils.GetServiceCertValidityPeriod(computeClient.GetMeshConfig()) },
		func() time.Duration { return utils.GetIngressGatewayCertValidityPeriod(computeClient.GetMeshConfig()) },
		checkInterval,
	)
}

// NewCertificateManagerFromMRC returns a new certificate manager.
func NewCertificateManagerFromMRC(ctx context.Context, kubeClient kubernetes.Interface, kubeConfig *rest.Config,
	providerNamespace string, option Options, computeClient compute.Interface, checkInterval time.Duration) (*certificate.Manager, error) {
	if err := option.Validate(); err != nil {
		return nil, err
	}

	mrcClient := &MRCComposer{
		Interface: computeClient,
		MRCProviderGenerator: MRCProviderGenerator{
			kubeClient:      kubeClient,
			kubeConfig:      kubeConfig,
			KeyBitSize:      utils.GetCertKeyBitSize(computeClient.GetMeshConfig()),
			caExtractorFunc: getCA,
		},
	}
	// TODO(#4745): Remove after deprecating the osm.vault.token option.
	if vaultOption, ok := option.(VaultOptions); ok {
		mrcClient.MRCProviderGenerator.DefaultVaultToken = vaultOption.VaultToken
	}

	return certificate.NewManager(
		ctx,
		mrcClient,
		func() time.Duration { return utils.GetServiceCertValidityPeriod(computeClient.GetMeshConfig()) },
		func() time.Duration { return utils.GetIngressGatewayCertValidityPeriod(computeClient.GetMeshConfig()) },
		checkInterval,
	)
}

// GetCertIssuerForMRC returns a certificate.Issuer generated from the provided MRC.
func (c *MRCProviderGenerator) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, pem.RootCertificate, error) {
	p := mrc.Spec.Provider
	var issuer certificate.Issuer
	var err error
	switch {
	case p.Tresor != nil:
		issuer, err = c.getTresorOSMCertificateManager(mrc)
	case p.Vault != nil:
		issuer, err = c.getHashiVaultOSMCertificateManager(mrc)
	case p.CertManager != nil:
		issuer, err = c.getCertManagerOSMCertificateManager(mrc)
	default:
		return nil, nil, fmt.Errorf("Unknown certificate provider: %+v", p)
	}

	if err != nil {
		return nil, nil, err
	}

	ca, err := c.caExtractorFunc(issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating init cert: %w", err)
	}

	return issuer, ca, nil
}

// getTresorOSMCertificateManager returns a certificate manager instance with Tresor as the certificate provider
func (c *MRCProviderGenerator) getTresorOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, error) {
	var err error
	var rootCert *certificate.Certificate

	// This part synchronizes CA creation using the inherent atomicity of kubernetes API backend
	// Assuming multiple instances of Tresor are instantiated at the same time, only one of them will
	// succeed to issue a "Create" of the secret. All other Creates will fail with "AlreadyExists".
	// Regardless of success or failure, all instances can proceed to load the same CA.
	rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		return nil, errors.New("failed to create new Certificate Authority with cert issuer tresor")
	}

	if rootCert.GetPrivateKey() == nil {
		return nil, errors.New("root cert does not have a private key")
	}

	rootCert, err = k8storage.GetCertificateFromSecret(mrc.Namespace, mrc.Spec.Provider.Tresor.CA.SecretRef.Name, rootCert, c.kubeClient)
	if err != nil {
		return nil, fmt.Errorf("failed to synchronize certificate on Secrets API : %w", err)
	}

	if rootCert.GetPrivateKey() == nil {
		return nil, fmt.Errorf("root cert does not have a private key: %w", certificate.ErrInvalidCertSecret)
	}

	tresorClient, err := tresor.New(
		rootCert,
		rootCertOrganization,
		c.KeyBitSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate Tresor as a Certificate Manager: %w", err)
	}

	return tresorClient, nil
}

// getHashiVaultOSMCertificateManager returns a certificate manager instance with Hashi Vault as the certificate provider
func (c *MRCProviderGenerator) getHashiVaultOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, error) {
	provider := mrc.Spec.Provider.Vault

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", provider.Protocol, provider.Host, provider.Port)

	// If the DefaultVaultToken is empty, query Vault token secret
	var err error
	vaultToken := c.DefaultVaultToken
	if vaultToken == "" {
		log.Debug().Msgf("Attempting to get Vault token from secret %s", provider.Token.SecretKeyRef.Name)
		vaultToken, err = getHashiVaultOSMToken(&provider.Token.SecretKeyRef, c.kubeClient)
		if err != nil {
			return nil, err
		}
	}

	vaultClient, err := vault.New(
		vaultAddr,
		vaultToken,
		provider.Role,
	)
	if err != nil {
		return nil, fmt.Errorf("error instantiating Hashicorp Vault as a Certificate Manager: %w", err)
	}

	return vaultClient, nil
}

// getHashiVaultOSMToken returns the Hashi Vault token from the secret specified in the provided secret key reference
func getHashiVaultOSMToken(secretKeyRef *v1alpha2.SecretKeyReferenceSpec, kubeClient kubernetes.Interface) (string, error) {
	tokenSecret, err := kubeClient.CoreV1().Secrets(secretKeyRef.Namespace).Get(context.TODO(), secretKeyRef.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error retrieving Hashi Vault token secret %s/%s: %w", secretKeyRef.Namespace, secretKeyRef.Name, err)
	}

	token, ok := tokenSecret.Data[secretKeyRef.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in Hashi Vault token secret %s/%s", secretKeyRef.Key, secretKeyRef.Namespace, secretKeyRef.Name)
	}

	return string(token), nil
}

// getCertManagerOSMCertificateManager returns a certificate manager instance with cert-manager as the certificate provider
func (c *MRCProviderGenerator) getCertManagerOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, error) {
	provider := mrc.Spec.Provider.CertManager
	client, err := cmversionedclient.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to build cert-manager client set: %w", err)
	}

	cmClient, err := certmanager.New(
		client,
		mrc.Namespace,
		cmmeta.ObjectReference{
			Name:  provider.IssuerName,
			Kind:  provider.IssuerKind,
			Group: provider.IssuerGroup,
		},
		c.KeyBitSize,
	)
	if err != nil {
		return nil, fmt.Errorf("error instantiating Jetstack cert-manager client: %w", err)
	}

	return cmClient, nil
}

package providers

import (
	"fmt"
	"time"

	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/castorage/k8s"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/certmanager"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/certificate/providers/vault"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

var getCA func(certificate.Issuer) (pem.RootCertificate, error) = func(i certificate.Issuer) (pem.RootCertificate, error) {
	cert, err := i.IssueCertificate("init-cert", 1*time.Second)
	if err != nil {
		return nil, err
	}

	return cert.GetIssuingCA(), nil
}

// NewCertificateManager returns a new certificate manager, with an MRC compat client.
// TODO(4502): Use an informer behind a feature flag.
func NewCertificateManager(kubeClient kubernetes.Interface, kubeConfig *rest.Config, cfg configurator.Configurator,
	providerNamespace string, options Options, msgBroker *messaging.Broker) (*certificate.Manager, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	// TODO(4502): Switch the compat client to an informer. Might need another struct to compose the informer and
	// provider generator.
	mrcClient := &MRCCompatClient{
		MRCProviderGenerator: MRCProviderGenerator{
			kubeClient:      kubeClient,
			kubeConfig:      kubeConfig,
			KeyBitSize:      cfg.GetCertKeyBitSize(),
			caExtractorFunc: getCA,
		},
		mrc: &v1alpha2.MeshRootCertificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "legacy-compat",
				Namespace: providerNamespace,
				Annotations: map[string]string{
					constants.MRCVersionAnnotation: "legacy-compat",
				},
			},
			Spec: v1alpha2.MeshRootCertificateSpec{
				TrustDomain: "cluster.local",
				Provider:    options.AsProviderSpec(),
			},
			// TODO(#4502): Detect if an actual MRC exists, and set the status accordingly.
			Status: v1alpha2.MeshRootCertificateStatus{
				State: constants.MRCStateActive,
			},
		},
	}

	// TODO(#4745): Remove after deprecating the osm.vault.token option.
	if vaultOption, ok := options.(VaultOptions); ok {
		mrcClient.MRCProviderGenerator.DefaultVaultToken = vaultOption.VaultToken
	}

	return certificate.NewManager(mrcClient, cfg.GetServiceCertValidityPeriod(), msgBroker)
}

// GetCertIssuerForMRC returns a certificate.Issuer generated from the provided MRC.
func (c *MRCProviderGenerator) GetCertIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, pem.RootCertificate, string, error) {
	p := mrc.Spec.Provider
	var issuer certificate.Issuer
	var id string
	var err error
	switch {
	case p.Tresor != nil:
		issuer, id, err = c.getTresorOSMCertificateManager(mrc)
	case p.Vault != nil:
		issuer, id, err = c.getHashiVaultOSMCertificateManager(mrc)
	case p.CertManager != nil:
		issuer, id, err = c.getCertManagerOSMCertificateManager(mrc)
	default:
		return nil, nil, "", fmt.Errorf("Unknown certificate provider: %+v", p)
	}

	if err != nil {
		return nil, nil, "", err
	}

	ca, err := c.caExtractorFunc(issuer)
	if err != nil {
		return nil, nil, "", fmt.Errorf("error generating init cert: %w", err)
	}

	return issuer, ca, id, nil
}

func getMRCID(mrc *v1alpha2.MeshRootCertificate) (string, error) {
	if mrc.Annotations == nil || mrc.Annotations[constants.MRCVersionAnnotation] == "" {
		return "", fmt.Errorf("no annotation found for MRC %s/%s, expected annotation %s", mrc.Namespace, mrc.Name, constants.MRCVersionAnnotation)
	}
	return mrc.Annotations[constants.MRCVersionAnnotation], nil
}

// getTresorOSMCertificateManager returns a certificate manager instance with Tresor as the certificate provider
func (c *MRCProviderGenerator) getTresorOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, string, error) {
	var err error
	var rootCert *certificate.Certificate

	// This part synchronizes CA creation using the inherent atomicity of kubernetes API backend
	// Assuming multiple instances of Tresor are instantiated at the same time, only one of them will
	// succeed to issue a "Create" of the secret. All other Creates will fail with "AlreadyExists".
	// Regardless of success or failure, all instances can proceed to load the same CA.
	rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err != nil {
		return nil, "", errors.New("Failed to create new Certificate Authority with cert issuer tresor")
	}

	if rootCert.GetPrivateKey() == nil {
		return nil, "", errors.New("Root cert does not have a private key")
	}

	rootCert, err = k8s.GetCertificateFromSecret(mrc.Namespace, mrc.Spec.Provider.Tresor.CA.SecretRef.Name, rootCert, c.kubeClient)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to synchronize certificate on Secrets API : %w", err)
	}

	if rootCert.GetPrivateKey() == nil {
		return nil, "", fmt.Errorf("Root cert does not have a private key: %w", certificate.ErrInvalidCertSecret)
	}

	tresorClient, err := tresor.New(
		rootCert,
		rootCertOrganization,
		c.KeyBitSize,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to instantiate Tresor as a Certificate Manager: %w", err)
	}

	id, err := getMRCID(mrc)
	if err != nil {
		return nil, "", err
	}
	return tresorClient, id, nil
}

// getHashiVaultOSMCertificateManager returns a certificate manager instance with Hashi Vault as the certificate provider
func (c *MRCProviderGenerator) getHashiVaultOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, string, error) {
	provider := mrc.Spec.Provider.Vault

	// A Vault address would have the following shape: "http://vault.default.svc.cluster.local:8200"
	vaultAddr := fmt.Sprintf("%s://%s:%d", provider.Protocol, provider.Host, provider.Port)
	// TODO(#4502): If the DefaultVaultToken is empty, query the mrc.provider.vault.token.secretRef.
	vaultClient, err := vault.New(
		vaultAddr,
		c.DefaultVaultToken,
		provider.Role,
	)
	if err != nil {
		return nil, "", fmt.Errorf("error instantiating Hashicorp Vault as a Certificate Manager: %w", err)
	}
	id, err := getMRCID(mrc)
	if err != nil {
		return nil, "", err
	}
	return vaultClient, id, nil
}

// getCertManagerOSMCertificateManager returns a certificate manager instance with cert-manager as the certificate provider
func (c *MRCProviderGenerator) getCertManagerOSMCertificateManager(mrc *v1alpha2.MeshRootCertificate) (certificate.Issuer, string, error) {
	provider := mrc.Spec.Provider.CertManager
	client, err := cmversionedclient.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to build cert-manager client set: %s", err)
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
		return nil, "", fmt.Errorf("error instantiating Jetstack cert-manager client: %w", err)
	}
	id, err := getMRCID(mrc)
	if err != nil {
		return nil, "", err
	}
	return cmClient, id, nil
}

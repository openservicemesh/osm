package keyvault

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	az "github.com/Azure/go-autorest/autorest/azure"

	"github.com/open-service-mesh/osm/pkg/azure"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor"
)

// NewCertificateManager creates a new CertManager from an existing Azure Key Vault.
func NewCertificateManager(validity time.Duration, keyVaultName string, azureAuthFile string, osmID string, caType CertificateAuthorityType) (certificate.Manager, error) {
	authorizer, err := azure.GetAuthorizerWithRetryForKeyVault(azureAuthFile, az.PublicCloud.KeyVaultDNSSuffix)
	if err != nil {
		log.Error().Err(err).Msgf("Error getting Azure Key Vault authorizer")
		return nil, err
	}

	// TODO(draychev): temporary until there's time to...
	cn := certificate.CommonName("OpenServiceMesh")
	ca, err := tresor.NewCA(cn, validity)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new Certificate Authority")
	}

	keyVaultClient := keyvault.New()
	keyVaultClient.Authorizer = authorizer
	keyVaultClient.PollingDuration = pollingDurationTimeout
	return &keyVault{
		caType:        caType,
		name:          fmt.Sprintf("%s's Azure Key Vault", osmID),
		osmID:         osmID,
		client:        &keyVaultClient,
		vaultURL:      getKeyVaultURL(keyVaultName),
		validity:      validity,
		announcements: make(chan interface{}),
		cache:         make(map[certificate.CommonName]certificate.Certificater),
		ca:            ca,
	}, nil
}

type keyVault struct {
	// name is the name of the certificate manager (unique commonName of the Azure Key Vault)
	name string

	// osmID is the unique commonName of the service mesh this Azure Key Vault is supporting
	osmID string

	// caType indicates the kind of Certificate Authority we are going to use.
	caType CertificateAuthorityType

	client        *keyvault.BaseClient
	vaultURL      string
	validity      time.Duration
	cache         map[certificate.CommonName]certificate.Certificater
	announcements chan interface{}
	ca            certificate.Certificater
}

// GetAnnouncementsChannel returns a channel, which is used to announce when changes have been made to the issued certificates.
func (kv *keyVault) GetAnnouncementsChannel() <-chan interface{} {
	return kv.announcements
}

// IssueCertificate issues a new certificate for the given Subject Common Name.
func (kv *keyVault) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	log.Info().Msgf("Issuing certificate for CN=%s", cn)
	if cert, ok := kv.cache[cn]; ok {
		log.Trace().Msgf("Found certificate in cache CN=%s; No need to reach out to Azure Key Vault", cn)
		return cert, nil
	}
	// TODO(draychev): remove the randomValue
	randomValue := strconv.Itoa(rand.Int())
	certName := kv.getCertificateName(certificate.CommonName(string(cn) + randomValue))
	log.Trace().Msgf("Certificate not in cache CN=%s; Fetching from Azure Key Vault", cn)
	cert, privKey, err := kv.getCertificate(certName)
	if err == nil {
		return Certificate{
			commonName: certName.String(),
			certChain:  cert,
			privateKey: privKey,
		}, nil
	}

	if err == errCertNotFound {
		log.Trace().Err(err).Msgf("Cert CN=%s does not exist in Azure Key Vault", cn)
	} else {
		// TODO(draychev): proper return xx, err
		log.Fatal().Err(err).Msgf("Error creating certificate CN=%s in Azure Key Vault", cn)
	}

	// Convention is - the CA has the same commonName as the OSM ID
	caName := kv.osmID
	cert, privKey, err = kv.createCertificate(cn, caName)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating certificate")
		return nil, err
	}

	log.Info().Msgf("Certificate issued: %+v", cert)
	return nil, nil
}

func (kv *keyVault) ImportCertificate(name string, pemCert, pemKey []byte) error {
	b64EncodedCertificate := base64.StdEncoding.EncodeToString(pemCert)
	kv.importCertificate(&b64EncodedCertificate, name)
	return nil
}

func (kv *keyVault) GetName() string {
	return kv.name
}

func (kv *keyVault) getCertificate(certName certName) (cert []byte, privKey []byte, err error) {
	var getCert certGetter
	if kv.caType == WellKnownCertificateAuthority {
		getCert = kv.getCertificateWellKnownCA
	} else if kv.caType == CustomCertificateAuthority {
		getCert = kv.getCertificateCustomCA
	} else {
		return nil, nil, errors.New("unknown certificate authority type")
	}
	return getCert(certName)
}

func (kv *keyVault) getCertificateName(cn certificate.CommonName) certName {
	// Certificate commonName must match:  ^[0-9a-zA-Z-]+$
	// CN may have other characters - so we base64 encode it, and remove the trailing "="
	encodedCN := strings.Replace(base64.StdEncoding.EncodeToString([]byte(cn)), "=", "", -1)
	return certName(fmt.Sprintf("%s---%s", kv.osmID, encodedCN))
}

func (kv *keyVault) createCertificate(cn certificate.CommonName, caName string) ([]byte, []byte, error) {
	var createCert certCreator
	if kv.caType == WellKnownCertificateAuthority {
		createCert = kv.createCertificateWellKnownCA
	} else if kv.caType == CustomCertificateAuthority {
		createCert = kv.createCertificateCustomCA
	} else {
		return nil, nil, errUnknownCertificateAuthorityType
	}
	return createCert(cn, caName)
}

// GetIssuingCA implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetIssuingCA() []byte {
	if c.issuingCA == nil {
		log.Fatal().Msgf("No issuing CA available for cert %s", c.commonName)
		return nil
	}

	return c.issuingCA.GetCertificateChain()
}

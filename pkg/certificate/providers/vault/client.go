package vault

import (
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/certificate/providers/tresor/pem"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var log = logger.New("vault")

const (
	certificateField = "certificate"
	privateKeyField  = "private_key"
	issuingCAField   = "issuing_ca"
)

// NewCertManager implements certificate.Manager and wraps a Hashi Vault with methods to allow easy certificate issuance.
func NewCertManager(vaultAddr, token string, validity time.Duration) (*Client, error) {
	c := &Client{
		validity:      validity,
		announcements: make(chan interface{}),
		cache:         make(map[certificate.CommonName]Certificate),
	}
	config := api.DefaultConfig()
	config.Address = vaultAddr

	var err error
	if c.client, err = api.NewClient(config); err != nil {
		log.Fatal().Err(err).Msgf("Error creating Vault client without TLS at %s", vaultAddr)
		return nil, err
	}

	log.Info().Msgf("Created Vault client at %v", vaultAddr)

	c.client.SetToken(token)

	rootCert, err := c.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootExpiration)
	if err != nil {
		log.Error().Err(err).Msg("Error creating CA")
		return nil, err
	}

	c.ca = rootCert
	return c, nil
}

// IssueCertificate issues a certificate by leveraging the Hashi Vault client.
func (c *Client) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	secret, err := c.client.Logical().Write(getIssueURL(), getIssuanceData(cn, c.validity))
	if err != nil {
		log.Error().Err(err).Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}

	return newCert(cn, secret, time.Now().Add(c.validity)), nil
}

// GetAnnouncementsChannel returns a channel used by the Hashi Vault instance to signal when a certificate has been changed.
func (c *Client) GetAnnouncementsChannel() <-chan interface{} {
	return c.announcements
}

// Certificate implements certificate.Certificater
type Certificate struct {
	// The commonName of the certificate
	commonName certificate.CommonName

	// When the cert expires
	expiration time.Time

	// PEM encoded Certificate and Key (byte arrays)
	certChain  pem.Certificate
	privateKey pem.PrivateKey

	// Certificate authority signing this certificate.
	issuingCA pem.RootCertificate
}

// GetName returns the common name of the given certificate.
func (c Certificate) GetName() string {
	return c.commonName.String()
}

// GetCertificateChain returns the PEM encoded certificate.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey returns the PEM encoded private key of the given certificate.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetIssuingCA returns the root certificate signing the given cert.
func (c Certificate) GetIssuingCA() []byte {
	return c.issuingCA
}

func newCert(cn certificate.CommonName, secret *api.Secret, expiration time.Time) *Certificate {
	return &Certificate{
		commonName: cn,
		expiration: expiration,
		certChain:  pem.Certificate(secret.Data[certificateField].(string)),
		privateKey: []byte(secret.Data[privateKeyField].(string)),
		issuingCA:  pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}

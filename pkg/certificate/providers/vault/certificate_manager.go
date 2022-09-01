package vault

import (
	"fmt"
	"time"

	"github.com/hashicorp/vault/api"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/logger"
)

var log = logger.New("vault")

const (
	// The string value of the JSON key containing the certificate's Serial Number.
	// See: https://www.vaultproject.io/api-docs/secret/pki#sample-response-8
	serialNumberField = "serial_number"
	certificateField  = "certificate"
	privateKeyField   = "private_key"
	issuingCAField    = "issuing_ca"
	commonNameField   = "common_name"
	ttlField          = "ttl"
)

// New constructs a new certificate client using Vault's cert-manager
func New(vaultAddr, token, role string) (*CertManager, error) {
	if vaultAddr == "" {
		return nil, fmt.Errorf("vault address must not be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("vault token must not be empty")
	}
	if role == "" {
		return nil, fmt.Errorf("vault role must not be empty")
	}
	c := &CertManager{
		role: role,
	}
	config := api.DefaultConfig()
	config.Address = vaultAddr

	var err error
	if c.client, err = api.NewClient(config); err != nil {
		return nil, fmt.Errorf("error creating Vault CertManager without TLS at %s, got err: %w", vaultAddr, err)
	}
	log.Info().Msgf("Created Vault CertManager, with role=%q at %v", role, vaultAddr)

	c.client.SetToken(token)

	return c, nil
}

// IssueCertificate requests a new signed certificate from the configured Vault issuer.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
	secret, err := cm.client.Logical().Write(getIssueURL(cm.role), getIssuanceData(cn, validityPeriod))
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrIssuingCert)).
			Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}
	return newCert(cn, secret, time.Now().Add(validityPeriod)), nil
}

func newCert(cn certificate.CommonName, secret *api.Secret, expiration time.Time) *certificate.Certificate {
	return &certificate.Certificate{
		CommonName:   cn,
		SerialNumber: certificate.SerialNumber(secret.Data[serialNumberField].(string)),
		Expiration:   expiration,
		CertChain:    pem.Certificate(secret.Data[certificateField].(string)),
		PrivateKey:   []byte(secret.Data[privateKeyField].(string)),
		IssuingCA:    pem.RootCertificate(secret.Data[issuingCAField].(string)),
		TrustedCAs:   pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}
}

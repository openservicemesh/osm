package vault

import (
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/constants"
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

	checkCertificateExpirationInterval = 5 * time.Second
	decade                             = 8765 * time.Hour
)

// NewProvider implements certificate.Manager and wraps a Hashi Vault with methods to allow easy certificate issuance.
func NewProvider(vaultAddr string, token string, role string) (*Provider, error) {
	p := &Provider{
		role: vaultRole(role),
	}

	config := api.DefaultConfig()
	config.Address = vaultAddr

	var err error
	if p.client, err = api.NewClient(config); err != nil {
		return nil, errors.Errorf("Error creating Vault CertManager without TLS at %s", vaultAddr)
	}

	log.Info().Msgf("Created Vault CertManager, with role=%q at %v", role, vaultAddr)

	p.client.SetToken(token)

	return p, nil
}

func (p *Provider) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {

	issuingCA, serialNumber, err := p.GetIssuingCA(p.IssueCertificate)
	if err != nil {
		return nil, err
	}

	ca := &certificate.Certificate{
		CommonName:   constants.CertificationAuthorityCommonName,
		SerialNumber: serialNumber,
		Expiration:   time.Now().Add(decade),
		CertChain:    issuingCA,
		IssuingCA:    issuingCA,
	}

	// Instantiating a new certificate rotation mechanism will start a goroutine for certificate rotation.
	certificate.New(c).Start(checkCertificateExpirationInterval)

	secret, err := p.client.Logical().Delete().v.Logical().Write(getIssueURL(cm.role).String(), getIssuanceData(cn, validityPeriod))
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrIssuingCert)).
			Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}

	//todo(schristoff): expiration
	return &certificate.Certificate{
		CommonName:   cn,
		SerialNumber: certificate.SerialNumber(secret.Data[serialNumberField].(string)),
		// Expiration:   time.Now() + validityPeriod,
		CertChain:  pem.Certificate(secret.Data[certificateField].(string)),
		PrivateKey: []byte(secret.Data[privateKeyField].(string)),
		IssuingCA:  pem.RootCertificate(secret.Data[issuingCAField].(string)),
	}, nil
}

//todo(schristoff): is this needed?
func (p *Provider) getIssuingCA(issue func(certificate.CommonName, time.Duration) (*certificate.Certificate, error)) ([]byte, certificate.SerialNumber, error) {
	// Create a temp certificate to determine the public part of the issuing CA
	cert, err := issue("localhost", decade)
	if err != nil {
		return nil, "", err
	}

	issuingCA := cert.GetIssuingCA()

	// We are not going to need this certificate - remove it
	p.ReleaseCertificate(cert.GetCommonName())

	return issuingCA, cert.GetSerialNumber(), err
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (p *Provider) ReleaseCertificate(cn certificate.CommonName) {
	// TODO(draychev): implement Hashicorp Vault delete-cert API here: https://github.com/openservicemesh/osm/issues/2068
	secret, err := p.client.Logical().Delete().v.Logical().Write(getIssueURL(cm.role).String(), getIssuanceData(cn, validityPeriod))
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrIssuingCert)).
			Msgf("Error issuing new certificate for CN=%s", cn)
		return nil, err
	}
}

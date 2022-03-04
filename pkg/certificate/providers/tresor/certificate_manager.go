package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"time"

	"github.com/pkg/errors"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// NewCertManager creates a new CertManager with the passed CA and CA Private Key
func NewProvider(
	ca *certificate.Certificate,
	certificatesOrganization string,
	keySize int) (*Provider, error) {
	if ca == nil {
		return nil, errNoIssuingCA
	}

	return &Provider{
		// The root certificate signing all newly issued certificates
		ca:                       ca,
		certificatesOrganization: certificatesOrganization,
		keySize:                  keySize,
	}, nil
}

func (p *Provider) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
	if p.ca == nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidCA)).
			Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoIssuingCA
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, p.keySize)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingPrivateKey)).
			Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, errors.Wrap(err, errGeneratingPrivateKey.Error())
	}

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,

		DNSNames: []string{string(cn)},

		Subject: pkix.Name{
			CommonName:   string(cn),
			Organization: []string{p.certificatesOrganization},
		},
		NotBefore: now,
		NotAfter:  now.Add(validityPeriod),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	x509Root, err := certificate.DecodePEMCertificate(p.ca.GetCertificateChain())
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error decoding Root Certificate's PEM")
	}

	rsaKeyRoot, err := certificate.DecodePEMPrivateKey(p.ca.GetPrivateKey())
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMPrivateKey)).
			Msg("Error decoding Root Certificate's Private Key PEM ")
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, x509Root, &certPrivKey.PublicKey, rsaKeyRoot)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCert)).
			Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	certPEM, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingCertDERtoPEM)).
			Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingKeyDERtoPEM)).
			Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	cert := &certificate.Certificate{
		CommonName:   cn,
		SerialNumber: certificate.SerialNumber(serialNumber.String()),
		CertChain:    certPEM,
		PrivateKey:   privKeyPEM,
		IssuingCA:    pem.RootCertificate(p.ca.GetCertificateChain()),
		Expiration:   template.NotAfter,
	}

	log.Trace().Msgf("Created new certificate for SerialNumber=%s; validity=%+v; expires on %+v; serial: %x", serialNumber, validityPeriod, template.NotAfter, template.SerialNumber)

	return cert, nil
}

func (p *Provider) GetRootCertificate() (*certificate.Certificate, error) {
	return p.ca, nil
}

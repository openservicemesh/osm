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

// NewCA creates a new Certificate Authority.
func NewCA(cn certificate.CommonName, validityPeriod time.Duration, rootCertCountry, rootCertLocality, rootCertOrganization string) (*certificate.Certificate, error) {
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingSerialNumber.Error())
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   cn.String(),
			Country:      []string{rootCertCountry},
			Locality:     []string{rootCertLocality},
			Organization: []string{rootCertOrganization},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validityPeriod),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rsaKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingPrivateKey)).
			Msgf("Error generating key for CA for org %s", rootCertOrganization)
		return nil, err
	}

	// Self-sign the root certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &rsaKey.PublicKey, rsaKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingRootCert)).
			Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, errors.Wrap(err, errCreateCert.Error())
	}

	pemCert, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingCertDERtoPEM)).
			Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	pemKey, err := certificate.EncodeKeyDERtoPEM(rsaKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingKeyDERtoPEM)).
			Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	return &certificate.Certificate{
		CommonName:   certificate.CommonName(template.Subject.CommonName),
		SerialNumber: certificate.SerialNumber(serialNumber.String()),
		CertChain:    pemCert,
		IssuingCA:    pem.RootCertificate(pemCert),
		PrivateKey:   pemKey,
		Expiration:   template.NotAfter,
	}, nil
}

// NewCertificateFromPEM is a helper returning a *certificate.Certificate from the PEM components given.
func NewCertificateFromPEM(pemCert pem.Certificate, pemKey pem.PrivateKey) (*certificate.Certificate, error) {
	x509Cert, err := certificate.DecodePEMCertificate(pemCert)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error converting PEM cert to x509 to obtain serial number")
		return nil, err
	}

	return &certificate.Certificate{
		CommonName:   certificate.CommonName(x509Cert.Subject.CommonName),
		SerialNumber: certificate.SerialNumber(x509Cert.SerialNumber.String()),
		CertChain:    pemCert,
		IssuingCA:    pem.RootCertificate(pemCert),
		PrivateKey:   pemKey,
		Expiration:   x509Cert.NotAfter,
	}, nil
}

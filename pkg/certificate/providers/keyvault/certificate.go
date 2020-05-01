package keyvault

import (
	"crypto/x509"
)

// GetName implements certificate.Certificater and returns the CN of the cert.
func (c Certificate) GetName() string {
	return c.commonName
}

// GetCertificateChain implements certificate.Certificater and returns the certificate chain.
func (c Certificate) GetCertificateChain() []byte {
	return c.certChain
}

// GetPrivateKey implements certificate.Certificater and returns the private key of the cert.
func (c Certificate) GetPrivateKey() []byte {
	return c.privateKey
}

// GetRootCertificate implements certificate.Certificater and returns the root certificate for the given cert.
func (c Certificate) GetRootCertificate() *x509.Certificate {
	log.Error().Msg("CA cannot be retrieved when using Azure Key Vault")
	return nil
}

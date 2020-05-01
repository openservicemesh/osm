package keyvault

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/v7.0/keyvault"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/logger"
)

var (
	log = logger.New("azure-keyvault-client")
)

type client struct {
	client        *keyvault.BaseClient
	vaultURL      string
	announcements chan interface{}
}

// Certificate implements certificate.Certificater
type Certificate struct {
	commonName string
	certChain  []byte
	privateKey []byte
	issuingCA  certificate.Certificater
}

type certName string

func (c certName) String() string {
	return string(c)
}

// CertificateAuthorityType is an enum of the CA kinds
type CertificateAuthorityType int

const (
	// WellKnownCertificateAuthority is an enum indicating that we are going to use a well known CA (DigiCert etc.)
	WellKnownCertificateAuthority CertificateAuthorityType = iota + 1

	// CustomCertificateAuthority is an enum indicating we are using a self signed CA.
	CustomCertificateAuthority
)

type certGetter func(certName) (cert []byte, privKey []byte, err error)

type certCreator func(cn certificate.CommonName, caName string) (cert []byte, privKey []byte, err error)

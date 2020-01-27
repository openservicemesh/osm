package certificate

// CommonName is the Subject Common Name from a given SSL certificate.
type CommonName string

// Certificater is the interface declaring methods each Certificate object must have.
type Certificater interface {

	// GetName retrieves the name of the cerificate.
	GetName() string

	// GetCertificateChain retrieves the cert chain.
	GetCertificateChain() []byte

	// GetPrivateKey returns the private key.
	GetPrivateKey() []byte
}

// Manager is the interface declaring the methods for the Certificate Maneger.
type Manager interface {
	// IssueCertificate issues a new certificate.
	IssueCertificate(cn CommonName) (Certificater, error)

	// GetSecretsChangeAnnouncementChan returns a channel, which is used to announce when changes have been made to the issued certificates.
	GetSecretsChangeAnnouncementChan() <-chan interface{}
}

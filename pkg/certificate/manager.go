package certificate

type CertManager struct {
	announcements chan interface{}
}

func NewManager(stop chan struct{}) CertManager {
	return CertManager{
		announcements: make(chan interface{}),
	}
}

// IssueCertificate implements Manager interface
func (m CertManager) IssueCertificate(cn CommonName) (Certificater, error) {
	// TODO(draychev): implement this function (and remove the shim below)
	return newCertificate(cn)
}

func (m CertManager) GetSecretsChangeAnnouncementChan() <-chan interface{} {
	return m.announcements
}

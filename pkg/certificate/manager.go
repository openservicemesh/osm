package certificate

type CertManager struct {
	announcementsChan chan struct{}
}

func NewManager(stop chan struct{}) CertManager {
	return CertManager{
		announcementsChan: make(chan struct{}),
	}
}

// IssueCertificate implements Manager interface
func (m CertManager) IssueCertificate(cn CommonName) (Certificater, error) {
	// TODO(draychev): implement this function (and remove the shim below)
	return newCertificate(cn)
}

func (m CertManager) GetSecretsChangeAnnouncementChan() <-chan struct{} {
	return m.announcementsChan
}

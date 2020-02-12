package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/deislabs/smc/pkg/certificate"
)

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	if cert, exists := cm.cache[cn]; exists {
		return cert, nil
	}
	glog.Infof("[tresor] Issuing Certificate for CN=%s", cn)
	if cm.ca == nil || cm.caPrivKey == nil {
		return nil, errNoCA
	}
	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, errGeneratingPrivateKey.Error())
	}
	template, err := makeTemplate(string(cn), cm.org, cm.validity)
	if err != nil {
		return nil, err
	}
	certPEM, privKeyPEM, err := genCert(template, cm.ca, certPrivKey, cm.caPrivKey)
	if err != nil {
		return nil, err
	}
	cert := Certificate{
		name:       string(cn),
		certChain:  certPEM,
		privateKey: privKeyPEM,
	}
	cm.cache[cn] = cert
	return cert, nil
}

// GetSecretsChangeAnnouncementChan implements certificate.Manager and returns the channel on which the certificate manager announces changes made to certificates.
func (cm CertManager) GetSecretsChangeAnnouncementChan() <-chan interface{} {
	return cm.announcements
}

package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"reflect"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/open-service-mesh/osm/pkg/certificate"
)

const (
	rsaBits = 4096
)

type empty struct{}

var packageName = reflect.TypeOf(empty{}).PkgPath()

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	if cert, exists := cm.cache[cn]; exists {
		return cert, nil
	}
	log.Info().Msgf("[%s] Issuing Certificate for CN=%s", packageName, cn)
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
		ca:         cm.ca,
	}
	cm.cache[cn] = cert
	return cert, nil
}

// GetAnnouncementsChannel implements certificate.Manager and returns the channel on which the certificate manager announces changes made to certificates.
func (cm CertManager) GetAnnouncementsChannel() <-chan interface{} {
	return cm.announcements
}

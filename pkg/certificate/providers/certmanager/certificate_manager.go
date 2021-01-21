package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	cminformers "github.com/jetstack/cert-manager/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/configurator"
)

// IssueCertificate implements certificate.Manager and returns a newly issued certificate.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	start := time.Now()

	// Attempt to grab certificate from cache.
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}

	// Cache miss/needs rotation so issue new certificate.
	cert, err := cm.issue(cn, validityPeriod)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("It took %+v to issue certificate with SerialNumber=%s", time.Since(start), cert.GetSerialNumber())

	return cert, nil
}

// ReleaseCertificate is called when a cert will no longer be needed and should be removed from the system.
func (cm *CertManager) ReleaseCertificate(cn certificate.CommonName) {
	cm.deleteFromCache(cn)
}

// GetCertificate returns a certificate given its Common Name (CN)
func (cm *CertManager) GetCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	if cert := cm.getFromCache(cn); cert != nil {
		return cert, nil
	}
	return nil, fmt.Errorf("failed to find certificate with CN=%s", cn)
}

func (cm *CertManager) deleteFromCache(cn certificate.CommonName) {
	cm.cacheLock.RLock()
	delete(cm.cache, cn)
	cm.cacheLock.RUnlock()
}

func (cm *CertManager) getFromCache(cn certificate.CommonName) certificate.Certificater {
	cm.cacheLock.RLock()
	defer cm.cacheLock.RUnlock()
	if cert, exists := cm.cache[cn]; exists {
		log.Trace().Msgf("Certificate with SerialNumber=%s found in cache", cert.GetSerialNumber())
		if rotor.ShouldRotate(cert) {
			log.Trace().Msgf("Certificate with SerialNumber=%s found in cache but has expired", cert.GetSerialNumber())
			return nil
		}
		return cert
	}
	return nil
}

// RotateCertificate implements certificate.Manager and rotates an existing
// certificate. When a certificate is successfully created, garbage collect
// old CertificateRequests.
func (cm *CertManager) RotateCertificate(cn certificate.CommonName) (certificate.Certificater, error) {
	start := time.Now()

	newCert, err := cm.issue(cn, cm.cfg.GetServiceCertValidityPeriod())
	if err != nil {
		return newCert, err
	}

	cm.cacheLock.Lock()
	oldCert := cm.cache[cn]
	cm.cache[cn] = newCert
	cm.cacheLock.Unlock()
	cm.announcements <- announcements.Announcement{}

	log.Debug().Msgf("Rotated certificate (old SerialNumber=%s) with new SerialNumber=%s; took %+v", oldCert.GetSerialNumber(), newCert.GetSerialNumber(), time.Since(start))

	return newCert, nil
}

// GetRootCertificate returns the root certificate in PEM format and its expiration.
func (cm *CertManager) GetRootCertificate() (certificate.Certificater, error) {
	return cm.ca, nil
}

// ListCertificates lists all certificates issued
func (cm *CertManager) ListCertificates() ([]certificate.Certificater, error) {
	var certs []certificate.Certificater
	cm.cacheLock.RLock()
	for _, cert := range cm.cache {
		certs = append(certs, cert)
	}
	cm.cacheLock.RUnlock()
	return certs, nil
}

// GetAnnouncementsChannel returns a channel, which is used to announce when
// changes have been made to the issued certificates.
func (cm *CertManager) GetAnnouncementsChannel() <-chan announcements.Announcement {
	return cm.announcements
}

// certificaterFromCertificateRequest will construct a certificate.Certificater
// from a give CertificateRequest and private key.
func (cm *CertManager) certificaterFromCertificateRequest(cr *cmapi.CertificateRequest, privateKey []byte) (certificate.Certificater, error) {
	if cr == nil {
		return nil, nil
	}

	cert, err := certificate.DecodePEMCertificate(cr.Status.Certificate)
	if err != nil {
		return nil, err
	}

	return Certificate{
		commonName:   certificate.CommonName(cert.Subject.CommonName),
		serialNumber: certificate.SerialNumber(cert.SerialNumber.String()),
		expiration:   cert.NotAfter,
		certChain:    cr.Status.Certificate,
		privateKey:   privateKey,
		issuingCA:    cm.ca.GetIssuingCA(),
	}, nil
}

// issue will request a new signed certificate from the configured cert-manager
// issuer.
func (cm *CertManager) issue(cn certificate.CommonName, validityPeriod time.Duration) (certificate.Certificater, error) {
	duration := &metav1.Duration{
		Duration: validityPeriod,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		log.Error().Err(err).Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, fmt.Errorf("failed to generate private key for certificate with CN=%s: %s", cn, err)
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		log.Error().Err(err).Msgf("Error encoding private key for certificate with CN=%s", cn)
		return nil, err
	}

	csr := &x509.CertificateRequest{
		Version:            3,
		SignatureAlgorithm: x509.SHA512WithRSA,
		PublicKeyAlgorithm: x509.RSA,
		Subject: pkix.Name{
			CommonName: cn.String(),
		},
		DNSNames: []string{cn.String()},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csr, certPrivKey)
	if err != nil {
		return nil, fmt.Errorf("error creating x509 certificate request: %s", err)
	}

	csrPEM, err := certificate.EncodeCertReqDERtoPEM(csrDER)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate request DER to PEM CN=%s: %s", cn, err)
	}

	cr := &cmapi.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "osm-",
			Namespace:    cm.namespace,
		},
		Spec: cmapi.CertificateRequestSpec{
			Duration: duration,
			IsCA:     false,
			Usages: []cmapi.KeyUsage{
				cmapi.UsageKeyEncipherment, cmapi.UsageDigitalSignature,
			},
			Request:   csrPEM,
			IssuerRef: cm.issuerRef,
		},
	}

	cr, err = cm.client.Create(context.TODO(), cr, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("Created CertificateRequest %s/%s for CN=%s", cm.namespace, cr.Name, cn)

	// TODO: add timeout option instead of 60s hard coded.
	cr, err = cm.waitForCertificateReady(cr.Name, time.Second*60)
	if err != nil {
		return nil, err
	}

	cert, err := cm.certificaterFromCertificateRequest(cr, privKeyPEM)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := cm.client.Delete(context.TODO(), cr.Name, metav1.DeleteOptions{}); err != nil {
			log.Error().Err(err).Msgf("failed to delete CertificateRequest %s/%s", cm.namespace, cr.Name)
		}
	}()

	cm.cacheLock.Lock()
	defer cm.cacheLock.Unlock()
	cm.cache[cert.GetCommonName()] = cert

	return cert, nil
}

// NewCertManager will construct a new certificate.Certificater implemented
// using Jetstack's cert-manager,
func NewCertManager(
	ca certificate.Certificater,
	client cmversionedclient.Interface,
	namespace string,
	issuerRef cmmeta.ObjectReference,
	cfg configurator.Configurator,
) (*CertManager, error) {
	informerFactory := cminformers.NewSharedInformerFactory(client, time.Second*30)
	crLister := informerFactory.Certmanager().V1beta1().CertificateRequests().Lister().CertificateRequests(namespace)

	// TODO: pass through graceful shutdown
	informerFactory.Start(make(chan struct{}))

	cm := &CertManager{
		ca:            ca,
		cache:         make(map[certificate.CommonName]certificate.Certificater),
		announcements: make(chan announcements.Announcement),
		namespace:     namespace,
		client:        client.CertmanagerV1beta1().CertificateRequests(namespace),
		issuerRef:     issuerRef,
		crLister:      crLister,
		cfg:           cfg,
	}

	// Instantiating a new certificate rotation mechanism will start a goroutine for certificate rotation.
	rotor.New(cm).Start(checkCertificateExpirationInterval)

	return cm, nil
}

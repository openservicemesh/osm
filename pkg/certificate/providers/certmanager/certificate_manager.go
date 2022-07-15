package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmversionedclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	cminformers "github.com/jetstack/cert-manager/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// New will construct a new certificate client using Jetstack's cert-manager,
func New(
	client cmversionedclient.Interface,
	namespace string,
	issuerRef cmmeta.ObjectReference,
	keySize int) (*CertManager, error) {
	informerFactory := cminformers.NewSharedInformerFactory(client, time.Second*30)
	crLister := informerFactory.Certmanager().V1().CertificateRequests().Lister().CertificateRequests(namespace)

	// TODO: pass through graceful shutdown
	informerFactory.Start(make(chan struct{}))

	if keySize == 0 {
		return nil, errors.New("key bit size cannot be zero")
	}

	return &CertManager{
		namespace: namespace,
		client:    client.CertmanagerV1().CertificateRequests(namespace),
		issuerRef: issuerRef,
		crLister:  crLister,
		keySize:   keySize,
	}, nil
}

// certificateFromCertificateRequest will construct a certificate.Certificate
// from a given CertificateRequest and private key.
func (cm *CertManager) certificateFromCertificateRequest(cr *cmapi.CertificateRequest, privateKey []byte) (*certificate.Certificate, error) {
	if cr == nil {
		return nil, nil
	}

	cert, err := certificate.DecodePEMCertificate(cr.Status.Certificate)
	if err != nil {
		return nil, err
	}

	ca := cr.Status.CA
	if len(ca) == 0 {
		ca = cert.RawIssuer
	}

	if len(ca) == 0 {
		return nil, fmt.Errorf("CA not found in certificate request %s/%s", cr.Namespace, cr.Name)
	}

	return &certificate.Certificate{
		CommonName:   certificate.CommonName(cert.Subject.CommonName),
		SerialNumber: certificate.SerialNumber(cert.SerialNumber.String()),
		Expiration:   cert.NotAfter,
		CertChain:    cr.Status.Certificate,
		PrivateKey:   privateKey,
		IssuingCA:    pem.RootCertificate(ca),
		TrustedCAs:   pem.RootCertificate(ca),
	}, nil
}

// IssueCertificate will request a new signed certificate from the configured cert-manager issuer.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, validityPeriod time.Duration) (*certificate.Certificate, error) {
	duration := &metav1.Duration{
		Duration: validityPeriod,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, cm.keySize)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingPrivateKey)).
			Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, fmt.Errorf("failed to generate private key for certificate with CN=%s: %w", cn, err)
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingKeyDERtoPEM)).
			Msgf("Error encoding private key for certificate with CN=%s", cn)
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
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCertReq)).
			Msg("error creating certificate request")
		return nil, fmt.Errorf("error creating x509 certificate request: %w", err)
	}

	csrPEM, err := certificate.EncodeCertReqDERtoPEM(csrDER)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingCertDERtoPEM)).
			Msg("error encoding cert request DER to PEM")
		return nil, fmt.Errorf("failed to encode certificate request DER to PEM CN=%s: %w", cn, err)
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

	cert, err := cm.certificateFromCertificateRequest(cr, privKeyPEM)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := cm.client.Delete(context.TODO(), cr.Name, metav1.DeleteOptions{}); err != nil {
			log.Error().Err(err).Msgf("failed to delete CertificateRequest %s/%s", cm.namespace, cr.Name)
		}
	}()

	return cert, nil
}

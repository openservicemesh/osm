package certmanager

import (
	"fmt"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1beta1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

// NewRootCertificateFromPEM is a helper returning a certificate.Certificater
// from the PEM components given.
func NewRootCertificateFromPEM(pemCert pem.Certificate) (certificate.Certificater, error) {
	cert, err := certificate.DecodePEMCertificate(pemCert)
	if err != nil {
		return nil, fmt.Errorf("failed to decoded root certificate: %s", err)
	}

	return Certificate{
		commonName:   certificate.CommonName(cert.Subject.CommonName),
		serialNumber: certificate.SerialNumber(cert.SerialNumber.String()),
		certChain:    pemCert,
		expiration:   cert.NotAfter,
		issuingCA:    pem.RootCertificate(pemCert),
	}, nil
}

// WaitForCertificateRequestReady waits for the CertificateRequest resource to
// enter a Ready state.
func (cm *CertManager) waitForCertificateReady(name string, timeout time.Duration) (*cmapi.CertificateRequest, error) {
	var (
		cr  *cmapi.CertificateRequest
		err error
	)

	err = wait.PollImmediate(time.Second, timeout,
		func() (bool, error) {
			cr, err = cm.crLister.Get(name)
			if apierrors.IsNotFound(err) {
				log.Info().Msgf("Failed to find CertificateRequest %s/%s", cm.namespace, name)
				return false, nil
			}

			if err != nil {
				return false, fmt.Errorf("error getting CertificateRequest %s: %v", name, err)
			}

			isReady := certificateRequestHasCondition(cr, cmapi.CertificateRequestCondition{
				Type:   cmapi.CertificateRequestConditionReady,
				Status: cmmeta.ConditionTrue,
			})
			if !isReady {
				log.Info().Msgf("CertificateRequest not ready %s/%s: %+v",
					cm.namespace, name, cr.Status.Conditions)
			}

			return isReady, nil
		},
	)

	// return certificate even when error to use for debugging
	return cr, err
}

// certificateRequestHasCondition will return true if the given
// CertificateRequest has a condition matching the provided
// CertificateRequestCondition. Only the Type and Status field will be used in
// the comparison, meaning that this function will return 'true' even if the
// Reason, Message and LastTransitionTime fields do not match.
func certificateRequestHasCondition(cr *cmapi.CertificateRequest, c cmapi.CertificateRequestCondition) bool {
	if cr == nil {
		return false
	}
	existingConditions := cr.Status.Conditions
	for _, cond := range existingConditions {
		if c.Type == cond.Type && c.Status == cond.Status {
			if c.Reason == "" || c.Reason == cond.Reason {
				return true
			}
		}
	}
	return false
}

package certmanager

import (
	"fmt"
	"time"

	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

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
				return false, fmt.Errorf("error getting CertificateRequest %s: %w", name, err)
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

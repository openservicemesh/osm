package certificate

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
)

// UseCase is how the certificate is used. Each use case will have its own rotation mechanism, that is necessary to
// determine if we can propagate the rotation forward.
type UseCase string

const (
	// UseCaseValidatingWebhook maps to the certificate use case for the validating webhook.
	UseCaseValidatingWebhook UseCase = "validatingWebhook"

	// UseCaseMutatingWebhook maps to the certificate use case for the Mutating Webhook
	UseCaseMutatingWebhook UseCase = "mutatingWebhook"

	// UseCaseXDSControlPlane maps to the certificate use case for the XDS Control Plane
	UseCaseXDSControlPlane UseCase = "xdsControlPlane"

	// UseCaseSidecar maps to the certificate use case for the  sidecar service certs
	UseCaseSidecar UseCase = "sidecar"

	// UseCaseBootstrap maps to the certificate use case for the  Bootstrap secrets
	UseCaseBootstrap UseCase = "bootstrap"

	// UseCaseGateway maps to the certificate use case for the  Gateway ingress cert.
	UseCaseGateway UseCase = "gateway"
)

var (
	allUseCases = []UseCase{
		UseCaseValidatingWebhook,
		UseCaseMutatingWebhook,
		UseCaseXDSControlPlane,
		UseCaseSidecar,
		UseCaseBootstrap,
		UseCaseGateway,
	}
)

func (uc UseCase) String() string {
	return string(uc)
}

func (m *Manager) handleMRCEvent(event MRCEvent) error {
	mrc := event.MRC
	// TODO(5046): remove this first call to setIssuers.
	err := m.setIssuers(event.MRC)
	if err == nil {
		return nil
	}
	log.Err(err).Msg("error setting issuers on mrc")
	// TODO(5046): improve logging in this method.
	if shouldUpdateMRCComponentStatus(mrc) {
		if err := m.retryMRCUpdateOnConflict(mrc, m.updateMRCComponentStatus); err != nil {
			return err
		}
	} else if m.shouldUpdateState(mrc) {
		log.Info().Str("mrc", namespacedMRCName(mrc)).Msgf("updating MRC state")
		if err := m.retryMRCUpdateOnConflict(mrc, m.updateMRCState); err != nil {
			return err
		}
	} else if m.shouldSetIssuers(mrc) {
		log.Info().Str("mrc", namespacedMRCName(mrc)).Msgf("setting new certificate issuers on the Certificate Manager for MRC in stage %s", mrc.Status.State)
		if err := m.setIssuers(mrc); err != nil {
			return err
		}
	}
	return nil
}

// Component statuses determine if we're ready to transition the mrc to the next state.
// TODO(5046): add unit tests for this, and other methods in here.
func shouldUpdateMRCComponentStatus(mrc *v1alpha2.MeshRootCertificate) bool {
	if isTerminal(mrc) {
		return false
	}

	if len(mrc.Status.Conditions) == 0 {
		return false
	}

	transitionAfter := mrc.Status.TransitionAfter
	if transitionAfter == nil {
		return false
	}

	//The truncate is needed so we truncate the monotonic clock.
	return transitionAfter.Truncate(0).After(time.Now())
}

// shouldUpdateState (and conditions).
func (m *Manager) shouldUpdateState(mrc *v1alpha2.MeshRootCertificate) bool {
	if m.shouldEnsureIssuerForMRC(mrc) {
		return true
	}

	// TODO(5046): determine the status to we need to be at to be able to move forward.
	var status v1alpha2.MeshRootCertificateComponentStatus

	for _, useCase := range allUseCases {
		switch useCase {
		case UseCaseValidatingWebhook:
			if mrc.Status.ComponentStatuses.ValidatingWebhook != status {
				return false
			}
		case UseCaseMutatingWebhook:
			if mrc.Status.ComponentStatuses.MutatingWebhook != status {
				return false
			}
		case UseCaseXDSControlPlane:
			if mrc.Status.ComponentStatuses.XDSControlPlane != status {
				return false
			}
		case UseCaseSidecar:
			if mrc.Status.ComponentStatuses.Sidecar != status {
				return false
			}
		case UseCaseBootstrap:
			if mrc.Status.ComponentStatuses.Bootstrap != status {
				return false
			}
		case UseCaseGateway:
			if mrc.Status.ComponentStatuses.Gateway != status {
				return false
			}
		}
	}
	return true
}

// update the MRCComponentStatus
func (m *Manager) updateMRCComponentStatus(mrc *v1alpha2.MeshRootCertificate) error {
	// TODO(5046): determine the status to set the owned components to.
	var status v1alpha2.MeshRootCertificateComponentStatus

	for _, useCase := range m.ownedUseCases {
		switch useCase {
		case UseCaseValidatingWebhook:
			// validating, issuing unknown.
			mrc.Status.ComponentStatuses.ValidatingWebhook = status
		case UseCaseMutatingWebhook:
			mrc.Status.ComponentStatuses.MutatingWebhook = status
		case UseCaseXDSControlPlane:
			mrc.Status.ComponentStatuses.XDSControlPlane = status
		case UseCaseSidecar:
			mrc.Status.ComponentStatuses.Sidecar = status
		case UseCaseBootstrap:
			mrc.Status.ComponentStatuses.Bootstrap = status
		case UseCaseGateway:
			mrc.Status.ComponentStatuses.Gateway = status
		}
	}

	_, err := m.mrcClient.UpdateMeshRootCertificate(mrc)
	return err
}

// isTerminal checks if the MRC is in a terminal state.
func isTerminal(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO(5046): check if the mrc is in a terminal state.
	return true
}

// state and condition get updated together.
func (m *Manager) updateMRCState(mrc *v1alpha2.MeshRootCertificate) error {
	// TODO(5046): update the MRC state & conditions, and issue the update via the MRCClient.

	// If it's not in a terminal state, set the next transition time.
	// NOTE: consider that this isTerminal check will need to apply against the updated mrc state which should happen
	// above in this method.
	if isTerminal(mrc) {
		mrc.Status.TransitionAfter = nil
	} else {
		mrc.Status.TransitionAfter = &metav1.Time{Time: time.Now().Add(mrcDurationPerStage)}
		log.Debug().Str("mrc", namespacedMRCName(mrc)).Msgf("updated MRC TransitionAfter to %s", mrc.Status.TransitionAfter)
	}

	if m.shouldEnsureIssuerForMRC(mrc) {
		_, _, err := m.mrcClient.GetCertIssuerForMRC(mrc)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRetrievingCA)).Str("mrc", namespacedMRCName(mrc)).Msg("failed to retrieve CA on passive MRC creation")

			// CA not accepted
			setMRCCondition(mrc, constants.MRCConditionTypeAccepted, corev1.ConditionFalse, errorRetrievingCAReason, err.Error())
			setMRCCondition(mrc, constants.MRCConditionTypeIssuingRollout, corev1.ConditionFalse, notAcceptedIssuingReason, err.Error())
			setMRCCondition(mrc, constants.MRCConditionTypeValidatingRollout, corev1.ConditionFalse, notAcceptedValidatingReason, err.Error())
			mrc.Status.State = constants.MRCStateError
			_, err := m.mrcClient.UpdateMeshRootCertificate(mrc)
			return err
		}

		log.Debug().Str("mrc", namespacedMRCName(mrc)).Msg("successfully retrieved CA on passive MRC creation")

		// CA accepted
		setMRCCondition(mrc, constants.MRCConditionTypeAccepted, corev1.ConditionTrue, certificateAcceptedReason, "certificate accepted")
		setMRCCondition(mrc, constants.MRCConditionTypeIssuingRollout, corev1.ConditionFalse, passiveStateIssuingReason, "passive intent")
		setMRCCondition(mrc, constants.MRCConditionTypeValidatingRollout, corev1.ConditionFalse, passiveStateValidatingReason, "passive intent")
		mrc.Status.State = constants.MRCStatePending
		_, err = m.mrcClient.UpdateMeshRootCertificate(mrc)
		return err
	}

	_, err := m.mrcClient.UpdateMeshRootCertificate(mrc)
	return err
}

func (m *Manager) shouldSetIssuers(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO(5046): check the states, and if in some form of an active state, AND if the MRC is not already the existing
	// then we should set MRC, so return true
	return true
}

func (m *Manager) shouldEnsureIssuerForMRC(mrc *v1alpha2.MeshRootCertificate) bool {
	return m.leaderMode && mrc.Spec.Intent == constants.MRCIntentPassive && len(mrc.Status.Conditions) == 0
}

func (m *Manager) setIssuers(mrc *v1alpha2.MeshRootCertificate) error {
	client, ca, err := m.mrcClient.GetCertIssuerForMRC(mrc)
	if err != nil {
		return err
	}

	c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain, SpiffeEnabled: mrc.Spec.SpiffeEnabled}
	switch {
	case mrc.Status.State == constants.MRCStateActive:
		m.mu.Lock()
		m.signingIssuer = c
		m.validatingIssuer = c
		m.mu.Unlock()
	case mrc.Status.State == constants.MRCStateIssuingRollback || mrc.Status.State == constants.MRCStateIssuingRollout:
		m.mu.Lock()
		m.signingIssuer = c
		m.mu.Unlock()
	case mrc.Status.State == constants.MRCStateValidatingRollback || mrc.Status.State == constants.MRCStateValidatingRollout:
		m.mu.Lock()
		m.validatingIssuer = c
		m.mu.Unlock()
		// TODO(5046): let's not have a default case.. what are the explicit states that can result in this?
	default:
		m.mu.Lock()
		m.signingIssuer = c
		m.validatingIssuer = c
		m.mu.Unlock()
	}
	return nil
}

// retryMRCUpdateOnConflict updates an MRC using the specified function with RetryOnConflict
func (m *Manager) retryMRCUpdateOnConflict(mrc *v1alpha2.MeshRootCertificate, updateMRCFunc func(*v1alpha2.MeshRootCertificate) error) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		curMRC := m.mrcClient.GetMeshRootCertificate(mrc.Name)
		if curMRC == nil {
			return fmt.Errorf("failed to get MRC %s on update", namespacedMRCName(curMRC))
		}

		return updateMRCFunc(curMRC)
	})
}

func namespacedMRCName(mrc *v1alpha2.MeshRootCertificate) string {
	return fmt.Sprintf("%s/%s", mrc.Namespace, mrc.Name)
}

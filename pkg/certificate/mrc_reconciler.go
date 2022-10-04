package certificate

import (
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
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
		if err := m.updateMRCComponentStatus(mrc); err != nil {
			return err
		}
	} else if shouldUpdateState(mrc) {
		if err := m.updateMRCState(mrc); err != nil {
			return err
		}
	} else if m.shouldSetIssuers(mrc) {
		log.Info().Msgf("setting new certificate issuers on the Certificate Manager for MRC %s in stage %s", mrc.GetName(), mrc.Status.State)
		return m.setIssuers(mrc)
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
func shouldUpdateState(mrc *v1alpha2.MeshRootCertificate) bool {
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
	status := getNextStatus(mrc)

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

	return m.mrcClient.UpdateMeshRootCertificate(mrc)
}
func getNextStatus(mrc *v1alpha2.MeshRootCertificate) v1alpha2.MeshRootCertificateComponentStatus {
	// possible status transitions:
	// unknown -> validating
	// validating -> issuing
	// validating -> unused
	// issuing -> unused
	switch {
	case shouldChangeStatusToValidating(mrc):
		return constants.MRCComponentStatusValidating
	case shouldChangeStatusToIssuing(mrc):
		return constants.MRCComponentStatusIssuing
	case shouldChangeStatusToUnused(mrc):
		return constants.MRCComponentStatusUnused
	default:
		return constants.MRCComponentStatusUnknown
	}
}

func shouldChangeStatusToUnused(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO: implement check from issuing/validating to rollback
	return false
}

func shouldChangeStatusToIssuing(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO: implement check from validating to issuing
	return false
}

// checks if the given MRC meets the attributes to move the componentStatuses to Validating
func shouldChangeStatusToValidating(mrc *v1alpha2.MeshRootCertificate) bool {
	// checks if intent is passive
	if !strings.EqualFold(string(mrc.Spec.Intent), constants.MRCIntentPassive) {
		return false
	}
	// checks if Accepted condition has status true
	if !isConditionTrue(mrc, constants.MRCConditionTypeAccepted) {
		return false
	}

	// checks if ValidatingRollout condition has status false with reason Pending
	if isConditionFalse(mrc, constants.MRCConditionTypeValidatingRollout) && isReasonPending(mrc, constants.MRCConditionTypeValidatingRollout) {
		return true
	}
	return false
}

// returns true if the condition reason is Pending for the condtion type specified
func isReasonPending(mrc *v1alpha2.MeshRootCertificate, condType v1alpha2.MeshRootCertificateConditionType) bool {
	for _, condition := range mrc.Status.Conditions {
		if condition.Type == condType && strings.EqualFold(condition.Reason, constants.MRCConditionReasonPending) {
			return true
		}
	}
	return false
}

// returns true if the condition status is true for the condtion type specified
func isConditionTrue(mrc *v1alpha2.MeshRootCertificate, condType v1alpha2.MeshRootCertificateConditionType) bool {
	for _, condition := range mrc.Status.Conditions {
		if condition.Type == condType && strings.EqualFold(string(condition.Status), string(constants.MRCConditionStatusTrue)) {
			return true
		}
	}
	return false
}

// returns true if the condition status is false for the condtion type specified
func isConditionFalse(mrc *v1alpha2.MeshRootCertificate, condType v1alpha2.MeshRootCertificateConditionType) bool {
	for _, condition := range mrc.Status.Conditions {
		if condition.Type == condType && strings.EqualFold(string(condition.Status), string(constants.MRCConditionStatusFalse)) {
			return true
		}
	}
	return false
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
	}

	// TODO(5046): add the retry loop.
	return m.mrcClient.UpdateMeshRootCertificate(mrc)
}

func (m *Manager) shouldSetIssuers(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO(5046): check the states, and if in some form of an active state, AND if the MRC is not already the existing
	// then we should set MRC, so return true
	return true
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

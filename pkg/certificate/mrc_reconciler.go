package certificate

import (
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

func (uc UseCase) String() string {
	return string(uc)
}
func (m *Manager) handleMRCEvent(event MRCEvent) error {
	mrc := event.MRC
	// TODO(5046): improve logging in this method.
	if m.shouldUpdateMRCComponentStatus(mrc) {
		if err := m.updateMRCComponentStatus(mrc); err != nil {
			return err
		}
	} else if shouldUpdateMRCState(mrc) {
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
func (m *Manager) shouldUpdateMRCComponentStatus(mrc *v1alpha2.MeshRootCertificate) bool {
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

// update the MRCComponentStatus
func (m *Manager) updateMRCComponentStatus(mrc *v1alpha2.MeshRootCertificate) error {
	var state v1alpha2.MeshRootCertificateComponentStatus
	switch mrc.Status.State {
	case constants.MRCConditionTypeValidatingRollout, constants.MRCConditionTypeIssuingRollback:
		state = v1alpha2.Validating
	case constants.MRCConditionTypeIssuingRollout:
		state = v1alpha2.Issuing
	case constants.MRCConditionTypeValidatingRollback:
		state = v1alpha2.Unused
	}

	for _, useCase := range m.ownedUseCases {
		switch useCase {
		case UseCaseValidatingWebhook:
			// validating, issuing unknown.
			mrc.Status.ComponentStatuses.ValidatingWebhook = state
		case UseCaseMutatingWebhook:
			mrc.Status.ComponentStatuses.MutatingWebhook = state
		case UseCaseXDSControlPlane:
			mrc.Status.ComponentStatuses.XDSControlPlane = state
		case UseCaseSidecar:
			mrc.Status.ComponentStatuses.Sidecar = state
		case UseCaseBootstrap:
			mrc.Status.ComponentStatuses.Bootstrap = state
		case UseCaseGateway:
			mrc.Status.ComponentStatuses.Gateway = state
		}
	}
	return m.mrcClient.UpdateMeshRootCertificate(mrc)
}

// The state determines how the root cert is used for validating/issuing certs, if at all.
func shouldUpdateMRCState(mrc *v1alpha2.MeshRootCertificate) bool {
	// TODO(5046): add a check to determine if the MRC is in a terminal state.
	return mrc.Status.ComponentStatuses.ValidatingWebhook != "" &&
		mrc.Status.ComponentStatuses.MutatingWebhook != "" &&
		mrc.Status.ComponentStatuses.XDSControlPlane != "" &&
		mrc.Status.ComponentStatuses.Sidecar != "" &&
		mrc.Status.ComponentStatuses.Bootstrap != "" &&
		mrc.Status.ComponentStatuses.Gateway != ""
}

// isTerminal checks if the MRC is in a terminal state.
func isTerminal(mrc *v1alpha2.MeshRootCertificate) bool {
	return false
}

func (m *Manager) updateMRCState(mrc *v1alpha2.MeshRootCertificate) error {
	// TODO(5046): update the MRC state & conditions, and issue the update via the MRCClient.
	// add the retry loop as well.

	// If it's not in a terminal state, set the next transition time.
	if isTerminal(mrc) {
		mrc.Status.TransitionAfter = nil
	} else {
		mrc.Status.TransitionAfter = &metav1.Time{Time: time.Now().Add(mrcDurationPerStage)}
	}

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

	c := &issuer{Issuer: client, ID: mrc.Name, CertificateAuthority: ca, TrustDomain: mrc.Spec.TrustDomain}
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

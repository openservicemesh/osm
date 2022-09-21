package certificate

import (
	"time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ready              v1alpha2.MeshRootCertificateConditionType = "Ready"
	accepted           v1alpha2.MeshRootCertificateConditionType = "Accepted"
	issuingRollout     v1alpha2.MeshRootCertificateConditionType = "IssuingRollout"
	validatingRollout  v1alpha2.MeshRootCertificateConditionType = "ValidatingRollout"
	issuingRollback    v1alpha2.MeshRootCertificateConditionType = "IssuingRollback"
	validatingRollback v1alpha2.MeshRootCertificateConditionType = "ValidatingRollback"
)

const (
	trueStatus    v1alpha2.MeshRootCertificateConditionStatus = "True"
	falseStatus   v1alpha2.MeshRootCertificateConditionStatus = "False"
	unknownStatus v1alpha2.MeshRootCertificateConditionStatus = "Unknown"
)

type mrcConditionReason string

const (
	pendingReason mrcConditionReason = "Pending"

	// Accepted type
	certificateAcceptedReason mrcConditionReason = "CertificateAccepted"

	// ValidatingRollout type
	passiveStateValidatingReason      mrcConditionReason = "PassiveState"
	passivelyInUseForValidationReason mrcConditionReason = "CertificatePassivelyInUseForValidation"

	// IssuingRollout type
	passiveStateIssuingReason      mrcConditionReason = "PassiveState"
	passivelyInUseForIssuingReason mrcConditionReason = "CertificatePassivelyInUseForIssuing"

	// IssuingRollback type
	isNotReadyIssuingReason      mrcConditionReason = "IsNotReady"
	passiveStateCertIssuedReason mrcConditionReason = "PassiveStateAndCertIsBeingIssued"
	noLongerIssuingReason        mrcConditionReason = "NoLongerIssuing"

	// ValidatingRollback type
	isNotReadyValidatingReason    mrcConditionReason = "IsNotReady"
	passiveStateCertTrustedReason mrcConditionReason = "PassiveStateAndCertIsTrusted"
	noLongerValidatingReason      mrcConditionReason = "NoLongerValidating"

	// Ready type
	rotationCompleteReason mrcConditionReason = "RotationComplete"
)

func setMRCCondition(mrc *v1alpha2.MeshRootCertificate, conditionType v1alpha2.MeshRootCertificateConditionType, conditionStatus v1alpha2.MeshRootCertificateConditionStatus, conditionReason mrcConditionReason, message string) {
	newCondition := v1alpha2.MeshRootCertificateCondition{
		Type:    conditionType,
		Status:  conditionStatus,
		Reason:  string(conditionReason),
		Message: message,
	}

	now := metav1.NewTime(time.Now())
	newCondition.LastTransitionTime = &now

	for i, condition := range mrc.Status.Conditions {
		if condition.Type != newCondition.Type {
			continue
		}

		// only update transition time on status change
		if condition.Status == newCondition.Status {
			newCondition.LastTransitionTime = condition.LastTransitionTime
		} else {
			log.Debug().Str("mrc", mrc.Name).Msgf("updating last transition time for condition type %s", condition.Type)
		}

		// update condition
		mrc.Status.Conditions[i] = newCondition
		log.Debug().Str("mrc", mrc.Name).Msgf("updated existing condition of type %s", condition.Type)
		return
	}

	// condition type not found, append it to condition list
	mrc.Status.Conditions = append(mrc.Status.Conditions, newCondition)
	log.Debug().Str("mrc", mrc.Name).Msgf("added new condition of type %s", conditionType)
}

func getMRCCondition(mrc *v1alpha2.MeshRootCertificate, conditionType v1alpha2.MeshRootCertificateConditionType) *v1alpha2.MeshRootCertificateCondition {
	for _, condition := range mrc.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// mrcHasCondition returns true if the given MRC has a condition matching the specified condition
// Only the type, status, and reason are used in comparison
func mrcHasCondition(mrc *v1alpha2.MeshRootCertificate, condition v1alpha2.MeshRootCertificateCondition) bool {
	for _, cond := range mrc.Status.Conditions {
		if cond.Type == condition.Type &&
			cond.Status == condition.Status &&
			cond.Reason == condition.Reason {
			return true
		}
	}
	return false
}

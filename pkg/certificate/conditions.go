package certificate

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

// MRC condition reasons
const (
	// Accepted type
	certificateAcceptedReason = "CertificateAccepted"
	errorRetrievingCAReason   = "ErrorRetrievingCA"

	// ValidatingRollout type
	passiveStateValidatingReason      = "PassiveState"
	passivelyInUseForValidatingReason = "CertificatePassivelyInUseForValidation"
	notAcceptedValidatingReason       = "NotAccepted"

	// IssuingRollout type
	passiveStateIssuingReason      = "PassiveState"
	passivelyInUseForIssuingReason = "CertificatePassivelyInUseForIssuing"
	notAcceptedIssuingReason       = "NotAccepted"

	// ValidatingRollback type
	isNotReadyValidatingReason    = "IsNotReady"
	passiveStateCertTrustedReason = "PassiveStateAndCertIsTrusted"
	noLongerValidatingReason      = "NoLongerValidating"
)

func setMRCCondition(mrc *v1alpha2.MeshRootCertificate, conditionType v1alpha2.MeshRootCertificateConditionType, conditionStatus corev1.ConditionStatus, conditionReason string, message string) {
	newCondition := v1alpha2.MeshRootCertificateCondition{
		Type:    conditionType,
		Status:  conditionStatus,
		Reason:  string(conditionReason),
		Message: message,
	}

	now := metav1.NewTime(time.Now())

	for i, condition := range mrc.Status.Conditions {
		if condition.Type != newCondition.Type {
			continue
		}

		// only update transition time on status change
		if condition.Status == newCondition.Status {
			newCondition.LastTransitionTime = condition.LastTransitionTime
		} else {
			newCondition.LastTransitionTime = &now
			log.Debug().Str("mrc", namespacedMRCName(mrc)).Msgf("updating last transition time for condition type %s", condition.Type)
		}

		// update condition
		mrc.Status.Conditions[i] = newCondition
		log.Debug().Str("mrc", namespacedMRCName(mrc)).Msgf("updated existing condition of type %s", condition.Type)
		return
	}

	// condition type not found, set the LastTransitionTime and append it to condition list
	newCondition.LastTransitionTime = &now
	mrc.Status.Conditions = append(mrc.Status.Conditions, newCondition)
	log.Debug().Str("mrc", namespacedMRCName(mrc)).Msgf("added new condition of type %s", conditionType)
}

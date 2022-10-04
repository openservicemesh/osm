package certificate

import (
	"testing"
	time "time"

	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestSetMRCCondition(t *testing.T) {
	now := metav1.NewTime(time.Now())

	testCases := []struct {
		name                       string
		mrc                        *v1alpha2.MeshRootCertificate
		expectedConditionsLen      int
		expectedLastTransitionTime *metav1.Time
		newCondition               v1alpha2.MeshRootCertificateCondition
	}{
		{
			name: "set new condition",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
					TrustDomain: "testDomain",
					Intent:      constants.MRCIntentPassive,
				},
			},
			expectedConditionsLen: 1,
			newCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    accepted,
				Status:  v1.ConditionTrue,
				Reason:  certificateAcceptedReason,
				Message: "test",
			},
		},
		{
			name: "update existing condition, no status change",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
					TrustDomain: "testDomain",
					Intent:      constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:               validatingRollback,
							Status:             v1.ConditionFalse,
							Reason:             isNotReadyValidatingReason,
							Message:            "test",
							LastTransitionTime: &now,
						},
					},
				},
			},
			expectedConditionsLen:      1,
			expectedLastTransitionTime: &now,
			newCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollback,
				Status:  v1.ConditionFalse,
				Reason:  noLongerValidatingReason,
				Message: "test",
			},
		},
		{
			name: "update status of existing condition",
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
					TrustDomain: "testDomain",
					Intent:      constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:    validatingRollout,
							Status:  v1.ConditionFalse,
							Reason:  passiveStateValidatingReason,
							Message: "test",
						},
					},
				},
			},
			expectedConditionsLen: 1,
			newCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollout,
				Status:  v1.ConditionTrue,
				Reason:  passivelyInUseForValidatingReason,
				Message: "test",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			setMRCCondition(tc.mrc, tc.newCondition.Type, tc.newCondition.Status, tc.newCondition.Reason, tc.newCondition.Message)

			a.Equal(tc.expectedConditionsLen, len(tc.mrc.Status.Conditions))
			for _, cond := range tc.mrc.Status.Conditions {
				if cond.Type != tc.newCondition.Type {
					continue
				}

				a.Equal(tc.newCondition.Status, cond.Status)
				a.Equal(tc.newCondition.Reason, cond.Reason)
				a.Equal(tc.newCondition.Message, cond.Message)

				if tc.expectedLastTransitionTime != nil {
					a.Equal(tc.expectedLastTransitionTime, cond.LastTransitionTime)
				}
			}
		})
	}
}

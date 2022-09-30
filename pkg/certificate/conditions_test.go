package certificate

import (
	"testing"
	time "time"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				Status:  trueStatus,
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
							Status:             falseStatus,
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
				Status:  falseStatus,
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
							Status:  falseStatus,
							Reason:  passiveStateValidatingReason,
							Message: "test",
						},
					},
				},
			},
			expectedConditionsLen: 1,
			newCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollout,
				Status:  trueStatus,
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

func TestGetMRCCondition(t *testing.T) {
	testCases := []struct {
		name                 string
		mrc                  *v1alpha2.MeshRootCertificate
		desiredConditionType v1alpha2.MeshRootCertificateConditionType
		expectedCondition    *v1alpha2.MeshRootCertificateCondition
	}{
		{
			name: "condition not found",
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
							Status:  falseStatus,
							Reason:  pendingReason,
							Message: "test2",
						},
					},
				},
			},
			desiredConditionType: accepted,
			expectedCondition:    nil,
		},
		{
			name: "condition found",
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
							Status:  falseStatus,
							Reason:  pendingReason,
							Message: "test2",
						},
					},
				},
			},
			desiredConditionType: validatingRollout,
			expectedCondition: &v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollout,
				Status:  falseStatus,
				Reason:  pendingReason,
				Message: "test2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			cond := getMRCCondition(tc.mrc, tc.desiredConditionType)
			a.Equal(tc.expectedCondition, cond)
		})
	}
}

func TestMRCHasCondition(t *testing.T) {
	testCases := []struct {
		name             string
		mrc              *v1alpha2.MeshRootCertificate
		desiredCondition v1alpha2.MeshRootCertificateCondition
		expectedReturn   bool
	}{
		{
			name: "has condition",
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
							Status:  falseStatus,
							Reason:  pendingReason,
							Message: "test",
						},
					},
				},
			},
			desiredCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollout,
				Status:  falseStatus,
				Reason:  pendingReason,
				Message: "test2",
			},
			expectedReturn: true,
		},
		{
			name: "has condition type, but not desired status",
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
							Status:  falseStatus,
							Reason:  pendingReason,
							Message: "test",
						},
					},
				},
			},
			desiredCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    validatingRollout,
				Status:  trueStatus,
				Reason:  passivelyInUseForValidatingReason,
				Message: "test",
			},
			expectedReturn: false,
		},
		{
			name: "does not have desired condition",
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
							Status:  falseStatus,
							Reason:  pendingReason,
							Message: "test",
						},
					},
				},
			},
			desiredCondition: v1alpha2.MeshRootCertificateCondition{
				Type:    accepted,
				Status:  trueStatus,
				Reason:  certificateAcceptedReason,
				Message: "test",
			},
			expectedReturn: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)
			hasCond := mrcHasCondition(tc.mrc, tc.desiredCondition)
			a.Equal(tc.expectedReturn, hasCond)
		})
	}
}

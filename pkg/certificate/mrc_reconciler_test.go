package certificate

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestShouldChangeStatusToValidating(t *testing.T) {
	testCases := []struct {
		name   string
		mrc    *v1alpha2.MeshRootCertificate
		expRes bool
	}{
		{
			name:   "should change status to validating",
			expRes: true,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusTrue,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusFalse,
							Reason: constants.MRCConditionReasonPending,
						},
					},
				},
			},
		},
		{
			name:   "intent is not passive",
			expRes: false,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: "active",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusTrue,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusFalse,
							Reason: constants.MRCConditionReasonPending,
						},
					},
				},
			},
		},
		{
			name:   "Accepted condition status is not true",
			expRes: false,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: "False",
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusFalse,
							Reason: constants.MRCConditionReasonPending,
						},
					},
				},
			},
		},
		{
			name:   "ValidatingRollout status is not false",
			expRes: false,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusTrue,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: "True",
							Reason: constants.MRCConditionReasonPending,
						},
					},
				},
			},
		},
		{
			name:   "ValidatingRollout reason is not Pending",
			expRes: false,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusTrue,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusFalse,
							Reason: "PassiveIntent",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			res := shouldChangeStatusToValidating(tc.mrc)
			assert.Equal(tc.expRes, res)
		})
	}
}

func TestGetNextStatus(t *testing.T) {
	testCases := []struct {
		name   string
		mrc    *v1alpha2.MeshRootCertificate
		expRes v1alpha2.MeshRootCertificateComponentStatus
	}{
		{
			name:   "initial status not set",
			expRes: constants.MRCComponentStatusUnknown,
			mrc:    &v1alpha2.MeshRootCertificate{},
		},
		{
			name:   "should change status to validating",
			expRes: constants.MRCComponentStatusValidating,
			mrc: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: constants.MRCIntentPassive,
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusTrue,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusFalse,
							Reason: constants.MRCConditionReasonPending,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			res := getNextStatus(tc.mrc)
			assert.Equal(tc.expRes, res)
		})
	}
}

package certificate

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
)

var activeMRC1 = &v1alpha2.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "mrc1",
	},
	Spec: v1alpha2.MeshRootCertificateSpec{
		Intent: v1alpha2.ActiveIntent,
	},
}

var passiveMRC1 = &v1alpha2.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "mrc1",
	},
	Spec: v1alpha2.MeshRootCertificateSpec{
		Intent: v1alpha2.PassiveIntent,
	},
}

var activeMRC2 = &v1alpha2.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "mrc2",
	},
	Spec: v1alpha2.MeshRootCertificateSpec{
		Intent: v1alpha2.ActiveIntent,
	},
}

var passiveMRC2 = &v1alpha2.MeshRootCertificate{
	ObjectMeta: metav1.ObjectMeta{
		Name: "mrc2",
	},
	Spec: v1alpha2.MeshRootCertificateSpec{
		Intent: v1alpha2.PassiveIntent,
	},
}

func TestHandleMRCEvent(t *testing.T) {
	testCases := []struct {
		name                 string
		mrcEvent             MRCEvent
		mrcList              []*v1alpha2.MeshRootCertificate
		expectedError        bool
		wantSigningIssuer    *issuer
		wantValidatingIssuer *issuer
	}{
		{
			name: "no issuers set when no MRCs found",
			mrcEvent: MRCEvent{
				MRCName: "my-mrc",
			},
			expectedError:        true,
			mrcList:              []*v1alpha2.MeshRootCertificate{},
			wantSigningIssuer:    nil,
			wantValidatingIssuer: nil,
		},
		{
			name: "no issuers set when more than 2 MRCs found",
			mrcEvent: MRCEvent{
				MRCName: "my-mrc",
			},
			expectedError: true,
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mrc",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						TrustDomain: "foo.bar.com",
						Intent:      v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						TrustDomain: "foo.bar.com",
						Intent:      v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						TrustDomain: "foo.bar.com",
						Intent:      v1alpha2.PassiveIntent,
					},
				},
			},
			wantSigningIssuer:    nil,
			wantValidatingIssuer: nil,
		},
		{
			name: "set issuers for single MRC",
			mrcEvent: MRCEvent{
				MRCName: "my-mrc",
			},
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-mrc",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						TrustDomain: "foo.bar.com",
						Intent:      v1alpha2.ActiveIntent,
					},
				},
			},
			wantSigningIssuer:    &issuer{Issuer: &fakeIssuer{id: "my-mrc"}, ID: "my-mrc", TrustDomain: "foo.bar.com", CertificateAuthority: pem.RootCertificate("rootCA")},
			wantValidatingIssuer: &issuer{Issuer: &fakeIssuer{id: "my-mrc"}, ID: "my-mrc", TrustDomain: "foo.bar.com", CertificateAuthority: pem.RootCertificate("rootCA")},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			m := &Manager{
				mrcClient: &fakeMRCClient{
					mrcList: tt.mrcList,
				},
			}

			err := m.handleMRCEvent(tt.mrcEvent)
			if !tt.expectedError {
				assert.NoError(err)
			} else {
				assert.Error(err)
			}

			assert.Equal(tt.wantSigningIssuer, m.signingIssuer)
			assert.Equal(tt.wantValidatingIssuer, m.validatingIssuer)
		})
	}
}

func TestValidateMRCIntents(t *testing.T) {
	tests := []struct {
		name   string
		mrc1   *v1alpha2.MeshRootCertificate
		mrc2   *v1alpha2.MeshRootCertificate
		result error
	}{
		{
			name:   "Two nil mrcs",
			mrc1:   nil,
			mrc2:   nil,
			result: ErrUnexpectedNilMRC,
		},
		{
			name:   "Single invalid mrc intent passive",
			mrc1:   passiveMRC1,
			mrc2:   passiveMRC1,
			result: ErrExpectedActiveMRC,
		},
		{
			name: "Invalid mrc intent foo",
			mrc1: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: "foo",
				},
			},
			mrc2: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.ActiveIntent,
				},
			},
			result: ErrUnknownMRCIntent,
		},
		{
			name: "Invalid mrc intent combination of passive and passive",
			mrc1: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.PassiveIntent,
				},
			},
			mrc2: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.PassiveIntent,
				},
			},
			result: ErrInvalidMRCIntentCombination,
		},
		{
			name: "Valid mrc intent combination of active and active",
			mrc1: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.ActiveIntent,
				},
			},
			mrc2: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.ActiveIntent,
				},
			},
			result: nil,
		},
		{
			name: "Valid mrc intent combination of active and passive",
			mrc1: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.ActiveIntent,
				},
			},
			mrc2: &v1alpha2.MeshRootCertificate{
				Spec: v1alpha2.MeshRootCertificateSpec{
					Intent: v1alpha2.PassiveIntent,
				},
			},
			result: nil,
		},
		{
			name:   "Single valid mrc intent active",
			mrc1:   activeMRC1,
			mrc2:   activeMRC1,
			result: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			err := ValidateMRCIntents(tt.mrc1, tt.mrc2)
			assert.Equal(tt.result, err)
		})
	}
}

func TestGetSigningAndValidatingMRCs(t *testing.T) {
	tests := []struct {
		name                  string
		mrcList               []*v1alpha2.MeshRootCertificate
		expectedError         error
		expectedSigningMRC    *v1alpha2.MeshRootCertificate
		expectedValidatingMRC *v1alpha2.MeshRootCertificate
	}{
		{
			name:          "no mrcs",
			mrcList:       []*v1alpha2.MeshRootCertificate{},
			expectedError: ErrNoMRCsFound,
		},
		{
			name: "more than 2 active and passive MRCs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc3",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
			},
			expectedError: ErrNumMRCExceedsMaxSupported,
		},
		{
			name: "single mrc is active",
			mrcList: []*v1alpha2.MeshRootCertificate{
				activeMRC1,
			},
			expectedSigningMRC:    activeMRC1,
			expectedValidatingMRC: activeMRC1,
		},
		{
			name: "single mrc is passive, expect err, validateMRCIntent fails",
			mrcList: []*v1alpha2.MeshRootCertificate{
				passiveMRC1,
			},
			expectedError: ErrExpectedActiveMRC,
		},
		{
			name: "mrc1 is active and mrc2 is passive",
			mrcList: []*v1alpha2.MeshRootCertificate{
				activeMRC1,
				passiveMRC2,
			},
			expectedSigningMRC:    activeMRC1,
			expectedValidatingMRC: passiveMRC2,
		},
		{
			name: "mrc1 is active and mrc2 is active",
			mrcList: []*v1alpha2.MeshRootCertificate{
				activeMRC1,
				activeMRC2,
			},
			expectedSigningMRC:    activeMRC1,
			expectedValidatingMRC: activeMRC2,
		},
		{
			name: "mrc1 is passive and mrc2 is active",
			mrcList: []*v1alpha2.MeshRootCertificate{
				passiveMRC1,
				activeMRC2,
			},
			expectedSigningMRC:    activeMRC2,
			expectedValidatingMRC: passiveMRC1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)

			signingMRC, validatingMRC, err := getSigningAndValidatingMRCs(tt.mrcList)
			if tt.expectedError != nil {
				assert.ErrorIs(err, tt.expectedError)
			} else {
				assert.NoError(err)
				assert.Equal(tt.expectedSigningMRC, signingMRC)
				assert.Equal(tt.expectedValidatingMRC, validatingMRC)
			}
		})
	}
}

func TestGetCertIssuers(t *testing.T) {
	mrc1Name := "mrc1"
	mrc2Name := "mrc2"
	tests := []struct {
		name                       string
		signingMRC                 *v1alpha2.MeshRootCertificate
		validatingMRC              *v1alpha2.MeshRootCertificate
		expectedSigningIssuerID    string
		expectedValidatingIssuerID string
		currentSigningIssuerID     string
		currentValidatingIssuerID  string
		expectedError              error
	}{
		{
			name:          "mrcs are nil",
			signingMRC:    nil,
			validatingMRC: nil,
			expectedError: ErrUnexpectedNilMRC,
		},
		{
			name:                       "same mrcs for signing and validating, issuer does not exist",
			signingMRC:                 activeMRC1,
			validatingMRC:              activeMRC1,
			expectedSigningIssuerID:    mrc1Name,
			expectedValidatingIssuerID: mrc1Name},
		{
			name:                       "same mrcs for signing and validating, issuer does exist as signingIsseur",
			signingMRC:                 activeMRC1,
			validatingMRC:              activeMRC1,
			currentSigningIssuerID:     mrc1Name,
			currentValidatingIssuerID:  mrc2Name,
			expectedSigningIssuerID:    mrc1Name,
			expectedValidatingIssuerID: mrc1Name,
		},
		{
			name:                       "mrc1 is active and mrc2 is passive and issuer for mrc2 does not exist",
			signingMRC:                 activeMRC1,
			validatingMRC:              passiveMRC2,
			currentSigningIssuerID:     mrc1Name,
			currentValidatingIssuerID:  mrc1Name,
			expectedSigningIssuerID:    mrc1Name,
			expectedValidatingIssuerID: mrc2Name,
		},
		{
			name:                       "mrc1 is active and mrc2 is passive and issuers exist",
			signingMRC:                 activeMRC1,
			validatingMRC:              passiveMRC2,
			currentSigningIssuerID:     mrc2Name,
			currentValidatingIssuerID:  mrc1Name,
			expectedSigningIssuerID:    mrc1Name,
			expectedValidatingIssuerID: mrc2Name,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)

			m := &Manager{}
			if tt.currentSigningIssuerID != "" {
				m.signingIssuer = &issuer{ID: tt.currentSigningIssuerID}
			}
			if tt.currentValidatingIssuerID != "" {
				m.validatingIssuer = &issuer{ID: tt.currentValidatingIssuerID}
			}

			signingIssuer, validatingIssuer, err := m.getCertIssuers(tt.signingMRC, tt.validatingMRC)
			if tt.expectedError != nil {
				assert.ErrorIs(err, tt.expectedError)
			} else {
				assert.NoError(err)
				assert.NotNil(signingIssuer)
				assert.Equal(tt.expectedSigningIssuerID, signingIssuer.ID)
				assert.NotNil(validatingIssuer)
				assert.Equal(tt.expectedValidatingIssuerID, validatingIssuer.ID)
			}
		})
	}
}

func TestFilterOutInactiveMRCs(t *testing.T) {
	tests := []struct {
		name            string
		mrcList         []*v1alpha2.MeshRootCertificate
		expectedMRCList []*v1alpha2.MeshRootCertificate
	}{
		{
			name:            "empty mrc list",
			mrcList:         []*v1alpha2.MeshRootCertificate{},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{},
		},
		{
			name: "mrc list with only inactive mrcs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.InactiveIntent,
					},
				},
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{},
		},
		{
			name: "mrc list with no inactive mrcs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.PassiveIntent,
					},
				},
			},
		},
		{
			name: "mrc list with no inactive mrcs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc2",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.InactiveIntent,
					},
				},
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mrc1",
					},
					Spec: v1alpha2.MeshRootCertificateSpec{
						Intent: v1alpha2.ActiveIntent,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)

			returnedMRCList := filterOutInactiveMRCs(tt.mrcList)
			assert.ElementsMatch(tt.expectedMRCList, returnedMRCList)
		})
	}
}

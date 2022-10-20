package certificate

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

const (
	testNamespace = "osm-system"
	trustDomain   = "foo.bar.com"
)

var (
	activeMRC1 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc1",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.ActiveIntent,
			TrustDomain: trustDomain,
		},
	}

	passiveMRC1 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc1",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.PassiveIntent,
			TrustDomain: trustDomain,
		},
	}

	inactiveMRC1 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc1",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.InactiveIntent,
			TrustDomain: trustDomain,
		},
	}

	activeMRC2 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc2",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.ActiveIntent,
			TrustDomain: trustDomain,
		},
	}

	passiveMRC2 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc2",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.PassiveIntent,
			TrustDomain: trustDomain,
		},
	}

	passiveMRC3 = &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mrc3",
			Namespace: testNamespace,
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Intent:      v1alpha2.PassiveIntent,
			TrustDomain: trustDomain,
		},
	}
)

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
				activeMRC1,
				activeMRC2,
				passiveMRC3,
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
				activeMRC1,
			},
			wantSigningIssuer:    &issuer{Issuer: &fakeIssuer{id: "mrc1"}, ID: "mrc1", TrustDomain: "foo.bar.com", CertificateAuthority: pem.RootCertificate("rootCA")},
			wantValidatingIssuer: &issuer{Issuer: &fakeIssuer{id: "mrc1"}, ID: "mrc1", TrustDomain: "foo.bar.com", CertificateAuthority: pem.RootCertificate("rootCA")},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			objects := make([]runtime.Object, len(tt.mrcList))
			for i := range tt.mrcList {
				objects[i] = tt.mrcList[i]
			}
			configClient := configFake.NewSimpleClientset(objects...)
			m := &Manager{
				mrcClient: &fakeMRCClient{
					configClient: configClient,
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
			name:   "two nil mrcs",
			mrc1:   nil,
			mrc2:   nil,
			result: ErrUnexpectedNilMRC,
		},
		{
			name:   "single invalid mrc intent passive",
			mrc1:   passiveMRC1,
			mrc2:   passiveMRC1,
			result: ErrExpectedActiveMRC,
		},
		{
			name: "invalid mrc intent foo",
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
			name: "invalid mrc intent combination of passive and passive",
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
			name: "valid mrc intent combination of active and active",
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
			name: "valid mrc intent combination of active and passive",
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
			name:   "single valid mrc intent active",
			mrc1:   activeMRC1,
			mrc2:   activeMRC1,
			result: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)
			err := validateMRCIntents(tt.mrc1, tt.mrc2)
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
				activeMRC1,
				passiveMRC2,
				passiveMRC3,
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
			expectedValidatingIssuerID: mrc1Name,
		},
		{
			name:                       "same mrcs for signing and validating, issuer does exist as signingIssuer",
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

			m := &Manager{mrcClient: &fakeMRCClient{}}
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
				inactiveMRC1,
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{},
		},
		{
			name: "mrc list with no inactive mrcs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				activeMRC1,
				passiveMRC2,
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{
				activeMRC1,
				passiveMRC2,
			},
		},
		{
			name: "mrc list with no inactive mrcs",
			mrcList: []*v1alpha2.MeshRootCertificate{
				inactiveMRC1,
				activeMRC2,
			},
			expectedMRCList: []*v1alpha2.MeshRootCertificate{
				activeMRC2,
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

func TestShouldUpdateIssuers(t *testing.T) {
	mrc1Name := "mrc1"
	mrc2Name := "mrc2"
	tests := []struct {
		name                      string
		expectedError             error
		expectedUpdate            bool
		signingMRC                *v1alpha2.MeshRootCertificate
		validatingMRC             *v1alpha2.MeshRootCertificate
		currentSigningIssuerID    string
		currentValidatingIssuerID string
	}{
		{
			name:          "nil MRC, expect error",
			expectedError: ErrUnexpectedNilMRC,
			signingMRC:    nil,
			validatingMRC: activeMRC1,
		},
		{
			name:           "2 MRCs in with active intents and issuers are not already set",
			expectedUpdate: true,
			signingMRC:     activeMRC2,
			validatingMRC:  activeMRC1,
		},
		{
			name:                      "2 MRCs with active intents and issuers are already set",
			expectedUpdate:            false,
			signingMRC:                activeMRC2,
			validatingMRC:             activeMRC1,
			currentSigningIssuerID:    mrc1Name,
			currentValidatingIssuerID: mrc2Name,
		},
		{
			name:                      "2 MRCs with active intents and issuers are already set to expected values",
			expectedUpdate:            false,
			signingMRC:                activeMRC2,
			validatingMRC:             activeMRC1,
			currentSigningIssuerID:    mrc2Name,
			currentValidatingIssuerID: mrc1Name,
		},
		{
			name:                      "1 active MRC and 1 passive MRC and issuers are already set to expected values",
			expectedUpdate:            false,
			signingMRC:                activeMRC2,
			validatingMRC:             passiveMRC1,
			currentSigningIssuerID:    mrc2Name,
			currentValidatingIssuerID: mrc1Name,
		},
		{
			name:                      "1 active MRC and issuers are already set to expected values",
			expectedUpdate:            false,
			signingMRC:                activeMRC1,
			validatingMRC:             activeMRC1,
			currentSigningIssuerID:    mrc1Name,
			currentValidatingIssuerID: mrc1Name,
		},
		{
			name:                      "1 active MRC and issuers are not set to expected values",
			expectedUpdate:            true,
			signingMRC:                activeMRC1,
			validatingMRC:             activeMRC1,
			currentSigningIssuerID:    mrc1Name,
			currentValidatingIssuerID: mrc2Name,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := tassert.New(t)

			m := &Manager{mrcClient: &fakeMRCClient{}}

			if tt.currentSigningIssuerID != "" {
				m.signingIssuer = &issuer{ID: tt.currentSigningIssuerID}
			}
			if tt.currentValidatingIssuerID != "" {
				m.validatingIssuer = &issuer{ID: tt.currentValidatingIssuerID}
			}

			shouldUpdate, err := m.shouldUpdateIssuers(tt.signingMRC, tt.validatingMRC)
			if tt.expectedError != nil {
				assert.ErrorIs(err, tt.expectedError)
			} else {
				assert.NoError(err)
			}

			assert.Equal(tt.expectedUpdate, shouldUpdate)
		})
	}
}

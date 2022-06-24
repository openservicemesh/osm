package providers

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name          string
		id            string
		namespace     string
		status        string
		mrc           *v1alpha2.MeshRootCertificate
		configClient  configClientset.Interface
		expectedError bool
	}{
		{
			name:      "update mrc not provided",
			id:        "mrc",
			namespace: "ns",
			status:    constants.MRCStateIssuingRollback,
			configClient: fakeConfig.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "mrc",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			expectedError: false,
		},
		{
			name:      "update mrc provided",
			id:        "mrc",
			namespace: "ns",
			status:    constants.MRCStateIssuingRollback,
			configClient: fakeConfig.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "mrc",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			mrc: &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "mrc",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			},
			expectedError: false,
		},
		{
			name:          "update mrc not found",
			id:            "mrc",
			namespace:     "ns",
			status:        constants.MRCStateIssuingRollback,
			configClient:  fakeConfig.NewSimpleClientset(),
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			mrcClient := &MRCComposer{configClient: tc.configClient}

			updatedMRC, err := mrcClient.Update(tc.id, tc.namespace, tc.status, tc.mrc)
			if tc.expectedError {
				assert.Error(err)
			} else {
				assert.NoError(err)
				assert.NotNil(updatedMRC)
				assert.Equal(tc.status, updatedMRC.Status.State)
			}
		})
	}
}

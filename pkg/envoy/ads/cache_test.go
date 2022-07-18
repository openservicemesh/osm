package ads

import (
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestGetProxyFromPod(t *testing.T) {
	assert := tassert.New(t)

	var (
		// Default fixtures for various test variables
		podName        = "pod"
		namespace      = "namespace"
		serviceAccount = "serviceAccount"
		validUUID      = uuid.New()
	)

	testCases := []struct {
		testCaseName string

		// Input
		pod *v1.Pod

		// Output check
		expectErr bool
	}{
		{
			testCaseName: "Pod with no label",
			expectErr:    true,
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					Labels:    make(map[string]string),
				},
				Spec: v1.PodSpec{
					ServiceAccountName: serviceAccount,
				},
			},
		},
		{
			testCaseName: "Pod with invalid UUID",
			expectErr:    true,
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: "invalid UUID",
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: serviceAccount,
				},
			},
		},
		{
			testCaseName: "Pod with valid UUID",
			expectErr:    false,
			pod: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					Labels: map[string]string{
						constants.EnvoyUniqueIDLabelName: validUUID.String(),
					},
				},
				Spec: v1.PodSpec{
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}

	for _, testCase := range testCases {
		proxyResult, err := GetProxyFromPod(testCase.pod)

		if testCase.expectErr {
			assert.Error(err)
		} else {
			assert.NotNil(proxyResult)
			assert.Equal(proxyResult.UUID, validUUID)
		}
	}
}

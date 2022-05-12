package mesh

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestIsMeshedPod(t *testing.T) {
	type test struct {
		pod      corev1.Pod
		isMeshed bool
	}

	testCases := []test{
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-1",
					Labels: map[string]string{
						// This test requires an actual UUID
						constants.EnvoyUniqueIDLabelName: uuid.New().String()},
				},
			},
			isMeshed: true,
		},
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-2",
				},
			},
			isMeshed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing if pod %s is meshed", tc.pod.Name), func(t *testing.T) {
			assert := tassert.New(t)

			isMeshed := ProxyLabelExists(tc.pod)
			assert.Equal(isMeshed, tc.isMeshed)
		})
	}
}

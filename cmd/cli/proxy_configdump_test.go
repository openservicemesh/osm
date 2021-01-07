package main

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestIsMeshedPod(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		pod      corev1.Pod
		isMeshed bool
	}

	testCases := []test{
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "pod-1",
					Labels: map[string]string{constants.EnvoyUniqueIDLabelName: "test"},
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
			isMeshed := isMeshedPod(tc.pod)
			assert.Equal(isMeshed, tc.isMeshed)
		})
	}
}

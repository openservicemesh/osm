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
			assert := tassert.New(t)

			isMeshed := isMeshedPod(tc.pod)
			assert.Equal(isMeshed, tc.isMeshed)
		})
	}
}

func TestAnnotateErrMsgWithPodNamespaceMsg(t *testing.T) {
	type test struct {
		errorMsg     string
		podName      string
		podNamespace string
		annotatedMsg string
	}

	podNamespaceActionableMsg := "Note: Use the flag --namespace to modify the intended pod namespace."

	testCases := []test{
		{
			"Proxy get command error for pod name [%s] in pod namespace [%s]",
			"test-pod-name",
			"test-namespace",
			"Proxy get command error for pod name [test-pod-name] in pod namespace [test-namespace]\n\n" + podNamespaceActionableMsg,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Testing annotated error message for pod name [%s] in pod namespace [%s]", tc.podName, tc.podNamespace), func(t *testing.T) {
			assert := tassert.New(t)

			assert.Equal(
				tc.annotatedMsg,
				annotateErrMsgWithPodNamespaceMsg(tc.errorMsg, tc.podName, tc.podNamespace).Error())
		})
	}
}

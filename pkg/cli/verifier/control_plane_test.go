package verifier

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestControlPlane(t *testing.T) {
	testNs := "test"

	testCases := []struct {
		name      string
		resources []runtime.Object
		expected  Result
	}{
		{
			name: "all control plane pods are running",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-2",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-injector-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-injector"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-bootstrap-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-bootstrap"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "control plane pod is not running",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodFailed, // not running
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-injector-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-injector"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-bootstrap-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-bootstrap"},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
			expected: Result{
				Status: Failure,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fake.NewSimpleClientset(tc.resources...)
			v := &ControlPlaneHealthVerifier{
				kubeClient: fakeClient,
				namespace:  testNs,
			}

			actual := v.Run()
			out := new(bytes.Buffer)
			Print(actual, out, 1)
			a.Equal(tc.expected.Status, actual.Status, out)
		})
	}
}

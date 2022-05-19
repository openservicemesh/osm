package verifier

import (
	"bytes"
	"errors"
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
		name               string
		resources          []runtime.Object
		controllerProbeErr error
		injectorProbeErr   error
		bootstrapProbeErr  error
		expected           Result
	}{
		{
			name: "all control plane pods are ready",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-2",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-injector-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-injector"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-bootstrap-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-bootstrap"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "control plane pods are not ready",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionFalse, // not ready
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-injector-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-injector"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionFalse, // init-container pending
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-bootstrap-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-bootstrap"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionFalse, // containers not ready
							},
						},
					},
				},
			},
			expected: Result{
				Status: Failure,
			},
		},
		{
			name: "control plane probes fails",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-controller-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-controller"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-injector-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-injector"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "osm-bootstrap-1",
						Namespace: testNs,
						Labels:    map[string]string{constants.AppLabel: "osm-bootstrap"},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							{
								Type:   corev1.PodScheduled,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.ContainersReady,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodInitialized,
								Status: corev1.ConditionTrue,
							},
							{
								Type:   corev1.PodReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			controllerProbeErr: errors.New("fake error"), // fake probe error
			injectorProbeErr:   errors.New("fake error"), // fake probe error
			bootstrapProbeErr:  errors.New("fake error"), // fake probe error
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
				kubeClient:       fakeClient,
				namespace:        testNs,
				controllerProber: fakeHTTPProber{err: tc.controllerProbeErr},
				injectorProber:   fakeHTTPProber{err: tc.injectorProbeErr},
				bootstrapProber:  fakeHTTPProber{err: tc.bootstrapProbeErr},
			}

			actual := v.Run()
			out := new(bytes.Buffer)
			Print(actual, out, 1)
			a.Equal(tc.expected.Status, actual.Status, out)
		})
	}
}

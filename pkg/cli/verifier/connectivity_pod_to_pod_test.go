package verifier

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestRun(t *testing.T) {
	testMeshName := "test"

	testCases := []struct {
		name      string
		resources []runtime.Object
		srcPod    types.NamespacedName
		dstPod    types.NamespacedName
		expected  Result
	}{
		{
			name: "pods have config to communicate",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
			},
			srcPod: types.NamespacedName{Namespace: "ns1", Name: "pod1"},
			dstPod: types.NamespacedName{Namespace: "ns2", Name: "pod2"},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "pod doesn't belong to monitored namespace",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2", // not monitored
					},
				},
			},
			srcPod: types.NamespacedName{Namespace: "ns1", Name: "pod1"},
			dstPod: types.NamespacedName{Namespace: "ns2", Name: "pod2"},
			expected: Result{
				Status: Failure,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fake.NewSimpleClientset(tc.resources...)
			v := &PodConnectivityVerifier{
				srcPod:     tc.srcPod,
				dstPod:     tc.dstPod,
				kubeClient: fakeClient,
				meshName:   testMeshName,
			}

			actual := v.Run()
			out := new(bytes.Buffer)
			Print(actual, out)
			a.Equal(tc.expected.Status, actual.Status, out)
		})
	}
}

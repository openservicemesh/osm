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

func TestEgressRun(t *testing.T) {
	const testMeshName = "test"

	testCases := []struct {
		name            string
		resources       []runtime.Object
		trafficAttr     TrafficAttribute
		srcPod          types.NamespacedName
		srcConfigGetter ConfigGetter
		expected        Result
	}{
		{
			name: "http egress to httpbin.org on port 80",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "curl",
						Namespace: "curl",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.EnvoyContainerName,
							},
						},
						ServiceAccountName: "curl",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "curl",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
			},
			trafficAttr: TrafficAttribute{
				SrcPod:       &types.NamespacedName{Namespace: "curl", Name: "curl"},
				AppProtocol:  "http",
				ExternalPort: 80,
				ExternalHost: "httpbin.org",
			},
			srcConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/curl_egress.json",
			},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "https egress to httpbin.org on port 443",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "curl",
						Namespace: "curl",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.EnvoyContainerName,
							},
						},
						ServiceAccountName: "curl",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "curl",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
			},
			trafficAttr: TrafficAttribute{
				SrcPod:       &types.NamespacedName{Namespace: "curl", Name: "curl"},
				AppProtocol:  "https",
				ExternalPort: 443,
				ExternalHost: "httpbin.org",
			},
			srcConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/curl_egress.json",
			},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "invalid tcp egress to port 100",
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "curl",
						Namespace: "curl",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.EnvoyContainerName,
							},
						},
						ServiceAccountName: "curl",
					},
				},
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "curl",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
			},
			trafficAttr: TrafficAttribute{
				SrcPod:       &types.NamespacedName{Namespace: "curl", Name: "curl"},
				AppProtocol:  "tcp",
				ExternalPort: 100,
			},
			srcConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/curl_egress.json",
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
			v := &EgressConnectivityVerifier{
				trafficAttr:        tc.trafficAttr,
				srcPodConfigGetter: tc.srcConfigGetter,
				kubeClient:         fakeClient,
				meshName:           testMeshName,
			}

			actual := v.Run()
			out := new(bytes.Buffer)
			Print(actual, out, 1)
			a.Equal(tc.expected.Status, actual.Status, out)
		})
	}
}

package verifier

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestRun(t *testing.T) {
	testMeshName := "test"

	testCases := []struct {
		name            string
		resources       []runtime.Object
		trafficAttr     TrafficAttribute
		srcPod          types.NamespacedName
		dstPod          types.NamespacedName
		srcConfigGetter ConfigGetter
		dstConfigGetter ConfigGetter
		expected        Result
	}{
		{
			name: "pods have config to communicate",
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
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin1",
						Namespace: "httpbin",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: constants.EnvoyContainerName,
							},
						},
						ServiceAccountName: "httpbin",
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
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "httpbin",
						Labels: map[string]string{
							constants.OSMKubeResourceMonitorAnnotation: testMeshName,
						},
					},
				},
				&corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin",
						Namespace: "httpbin",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{
								Port:       14001,
								TargetPort: intstr.FromInt(14001),
							},
						},
						// Must match service IP in outbound filter chain match in testdata/curl_permissive.json
						ClusterIP: "10.96.15.1",
					},
				},
				&corev1.Endpoints{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin",
						Namespace: "httpbin",
					},
					Subsets: []corev1.EndpointSubset{
						{
							Ports: []corev1.EndpointPort{
								{
									Port: 14001,
								},
							},
						},
					},
				},
			},
			trafficAttr: TrafficAttribute{
				SrcPod:      &types.NamespacedName{Namespace: "curl", Name: "curl"},
				DstPod:      &types.NamespacedName{Namespace: "httpbin", Name: "httpbin1"},
				DstService:  &types.NamespacedName{Namespace: "httpbin", Name: "httpbin"},
				AppProtocol: "http",
			},
			srcConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/curl_permissive.json",
			},
			dstConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/httpbin1_permissive.json",
			},
			expected: Result{
				Status: Success,
			},
		},
		{
			name: "pod doesn't have config to communicate",
			// Missing monitor annotation
			// Missing Envoy sidecar
			resources: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "ns1",
					},
					// No Envoy sidecar
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "ns2",
					},
					// No Envoy sidecar
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
			trafficAttr: TrafficAttribute{
				SrcPod: &types.NamespacedName{Namespace: "ns1", Name: "pod1"},
				DstPod: &types.NamespacedName{Namespace: "ns2", Name: "pod2"},
			},
			// Use configs that don't allow the pods to communicate.
			// Configs pertain to pods curl and httpbin, while this test uses
			// pods pod1 and pod2.
			srcConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/curl_permissive.json",
			},
			dstConfigGetter: fakeConfigGetter{
				configFilePath: "testdata/httpbin1_permissive.json",
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
			v := &PodConnectivityVerifier{
				trafficAttr:        tc.trafficAttr,
				srcPodConfigGetter: tc.srcConfigGetter,
				dstPodConfigGetter: tc.dstConfigGetter,
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

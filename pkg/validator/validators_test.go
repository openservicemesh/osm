package validator

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testclient "k8s.io/client-go/kubernetes/fake"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	fakePolicyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/informers"

	"github.com/openservicemesh/osm/pkg/messaging"
)

func newNsK8sObj(ns string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
}

func TestIngressBackendValidator(t *testing.T) {
	testCases := []struct {
		name                    string
		input                   *admissionv1.AdmissionRequest
		expResp                 *admissionv1.AdmissionResponse
		expErrStr               string
		existingIngressBackends []*policyv1alpha1.IngressBackend
	}{
		{
			name: "IngressBackend with valid protocol succeeds",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend with invalid protocol errors",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "invalid"
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Expected 'port.protocol' to be 'http' or 'https', got: invalid",
		},
		{
			name: "IngressBackend with valid TLS config succeeds",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "https",
									"port": {
										"number": 80,
										"protocol": "https"
									},
									"tls": {
										"skipClientCertificateValidation": true
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend with invalid mTLS config false",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "https",
									"port": {
										"number": 80,
										"protocol": "https"
									},
									"tls": {
										"skipClientCertificateValidation": false
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "HTTPS ingress with client certificate validation enabled must specify at least one 'AuthenticatedPrincipal` source",
		},
		{
			name: "IngressBackend with valid mTLS config succeeds",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "https",
									"port": {
										"number": 80,
										"protocol": "https"
									},
									"tls": {
										"skipClientCertificateValidation": false
									}
								}
							],
							"sources": [
								{
									"kind": "AuthenticatedPrincipal",
									"name": "client.ns.cluster.local"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend with valid source IPRange",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "IPRange",
									"name": "10.0.0.0/10"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend with invalid source IPRange errors",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "IPRange",
									"name": "invalid"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got 'invalid'",
		},
		{
			name: "IngressBackend with valid source kind Service",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "Service",
									"name": "foo",
									"namespace": "bar"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend with invalid source name for kind Service",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "Service",
									"namespace": "bar"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "'source.name' not specified for source kind Service",
		},
		{
			name: "IngressBackend with invalid source namespace for kind Service",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "Service",
									"name": "bar"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "'source.namespace' not specified for source kind Service",
		},
		{
			name: "IngressBackend with invalid source name for kind AuthenticatedPrincipal",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "AuthenticatedPrincipal",
									"name": ""
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "'source.name' not specified for source kind AuthenticatedPrincipal",
		},
		{
			name: "IngressBackend with invalid source kind",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "invalid"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Invalid 'source.kind' value specified. Must be one of: Service, AuthenticatedPrincipal, IPRange",
		},
		{
			name: "IngressBackend with invalid source kind",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							],
							"sources": [
								{
									"kind": "invalid"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Invalid 'source.kind' value specified. Must be one of: Service, AuthenticatedPrincipal, IPRange",
		},
		{
			name: "IngressBackend has duplicate backends",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"metadata": {
							"name": "test-1",
							"namespace": "default"
						},
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								},
								{
									"name": "test",
									"port": {
										"number": 80
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Duplicate backends detected with service name: test and port: 80",
		},
		{
			name: "success: IngressBackend has duplicate backend names but different ports",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"metadata": {
							"name": "test-1",
							"namespace": "default"
						},
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								},
								{
									"name": "test",
									"port": {
										"number": 8080,
										"protocol": "http"
									}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "IngressBackend conflicts with existing IngressBackend backends",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"metadata": {
							"name": "test-1",
							"namespace": "default"
						},
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							]
						}
					}
					`),
				},
			},
			existingIngressBackends: []*policyv1alpha1.IngressBackend{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "IngressBackend",
						APIVersion: "v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-2",
						Namespace: "default",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "test",
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			expResp:   nil,
			expErrStr: "error: duplicate backends detected\n[+] IngressBackend default/test-1 conflicts with default/test-2:\nBackend test specified in test-1 and test-2 conflicts\n\n",
		},
		{
			name: "success: IngressBackend conflicts with existing IngressBackend backends on different ports",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"metadata": {
							"name": "test-1",
							"namespace": "default"
						},
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							]
						}
					}
					`),
				},
			},
			existingIngressBackends: []*policyv1alpha1.IngressBackend{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "IngressBackend",
						APIVersion: "v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-2",
						Namespace: "default",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "test",
								Port: policyv1alpha1.PortSpec{
									Number:   8080,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "success: IngressBackend doesn't error on update",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "IngressBackend",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "IngressBackend",
						"metadata": {
							"name": "test-1",
							"namespace": "default"
						},
						"spec": {
							"backends": [
								{
									"name": "test",
									"port": {
										"number": 80,
										"protocol": "http"
									}
								}
							]
						}
					}
					`),
				},
			},
			existingIngressBackends: []*policyv1alpha1.IngressBackend{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "IngressBackend",
						APIVersion: "v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-1",
						Namespace: "default",
					},
					Spec: policyv1alpha1.IngressBackendSpec{
						Backends: []policyv1alpha1.BackendSpec{
							{
								Name: "test",
								Port: policyv1alpha1.PortSpec{
									Number:   80,
									Protocol: "http",
								},
							},
						},
					},
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			stop := make(chan struct{})
			defer close(stop)
			broker := messaging.NewBroker(stop)

			objects := make([]runtime.Object, len(tc.existingIngressBackends))
			for i := range tc.existingIngressBackends {
				objects[i] = tc.existingIngressBackends[i]
			}

			fakeClient := fakePolicyClientset.NewSimpleClientset(objects...)
			informerCollection, err := informers.NewInformerCollection("osm", stop,
				informers.WithPolicyClient(fakeClient),
				informers.WithKubeClient(testclient.NewSimpleClientset()),
			)
			assert.NoError(err)

			k8sClient := k8s.NewClient("osm-namespace", "osm-mesh-config", informerCollection, fakeClient, broker)
			policyClient := kube.NewClient(k8sClient)
			pv := &policyValidator{
				policyClient: policyClient,
			}

			// Block until we start getting ingressbackend updates
			// We only do this because the informerCollection doesn't have the
			// policy client's msgBroker eventhandler registered when it initially runs
			// and that leads to a race condition in tests
			if len(objects) > 0 {
				ticker := time.NewTicker(1 * time.Second)
				for {
					<-ticker.C
					list, err := fakeClient.PolicyV1alpha1().IngressBackends("default").List(context.Background(), metav1.ListOptions{})
					if err != nil {
						t.Logf("Error getting IngressBackends: %v", err)
						continue
					}
					if len(list.Items) > 0 {
						break
					}
				}
			}

			resp, err := pv.ingressBackendValidator(tc.input)
			assert.Equal(tc.expResp, resp)
			if tc.expErrStr == "" {
				// we expect a nil error
				assert.Nil(err)
			}
			if err != nil {
				assert.Equal(tc.expErrStr, err.Error())
			}
		})
	}
}

func TestEgressValidator(t *testing.T) {
	testCases := []struct {
		name      string
		input     *admissionv1.AdmissionRequest
		expResp   *admissionv1.AdmissionResponse
		expErrStr string
	}{
		{
			name: "matches.apiGroup is invalid",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "Egress",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"matches": [
								{
								"apiGroup": "invalid",
								"kind": "BadHttpRoute",
								"name": "Name"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Expected 'matches.apiGroup' to be one of [specs.smi-spec.io/v1alpha4 policy.openservicemesh.io/v1alpha1], got: invalid",
		},
		{
			name: "Egress with valid http route and API group passes",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "Egress",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"matches": [
								{
								"apiGroup": "specs.smi-spec.io/v1alpha4",
								"kind": "HTTPRouteGroup",
								"name": "Name"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "Egress with valid UpstreamTrafficSetting match",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "Egress",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"matches": [
								{
								"apiGroup": "policy.openservicemesh.io/v1alpha1",
								"kind": "UpstreamTrafficSetting",
								"name": "foo"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "Egress with multiple UpstreamTrafficSetting matches is invalid",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "Egress",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"matches": [
								{
								"apiGroup": "policy.openservicemesh.io/v1alpha1",
								"kind": "UpstreamTrafficSetting",
								"name": "foo"
								},
								{
								"apiGroup": "policy.openservicemesh.io/v1alpha1",
								"kind": "UpstreamTrafficSetting",
								"name": "bar"
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Cannot have more than 1 UpstreamTrafficSetting match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			resp, err := egressValidator(tc.input)
			assert.Equal(tc.expResp, resp)
			if err != nil {
				assert.Equal(tc.expErrStr, err.Error())
			}
		})
	}
}

func TestTrafficTargetValidator(t *testing.T) {
	testCases := []struct {
		name      string
		input     *admissionv1.AdmissionRequest
		expResp   *admissionv1.AdmissionResponse
		expErrStr string
	}{
		{
			name: "TrafficTarget namespace matches destination namespace",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha3",
					Version: "access.smi-spec.io",
					Kind:    "TrafficTarget",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha3",
						"kind": "TrafficTarget",
						"metadata": {
							"namespace": "destination-namespace"
						},
						"spec": {
							"destination": {
								"kind": "ServiceAccount",
								"name": "destination-name",
								"namespace": "destination-namespace"
							}
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "Traffic Target namespace does not match destination namespace",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha3",
					Version: "access.smi-spec.io",
					Kind:    "TrafficTarget",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha3",
						"kind": "TrafficTarget",
						"metadata": {
							"namespace": "another-namespace"
						},
						"spec": {
							"destination": {
								"kind": "ServiceAccount",
								"name": "destination-name",
								"namespace": "destination-namespace"
							}
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "The traffic target namespace (another-namespace) must match spec.Destination.Namespace (destination-namespace)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			resp, err := trafficTargetValidator(tc.input)
			assert.Equal(tc.expResp, resp)
			if err != nil {
				assert.Equal(tc.expErrStr, err.Error())
			}
		})
	}
}

func TestUpstreamTrafficSettingValidator(t *testing.T) {
	testNs := "test"
	testCases := []struct {
		name                            string
		input                           *admissionv1.AdmissionRequest
		expResp                         *admissionv1.AdmissionResponse
		expErrStr                       string
		existingUpstreamTrafficSettings []*policyv1alpha1.UpstreamTrafficSetting
	}{
		{
			name: "UpstreamTrafficSetting with unique host",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local"
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "UpstreamTrafficSetting with duplicate host",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local"
						}
					}
					`),
				},
			},
			existingUpstreamTrafficSettings: []*policyv1alpha1.UpstreamTrafficSetting{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "UpstreamTrafficSetting",
						APIVersion: "policy.openservicemesh.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin1",
						Namespace: testNs,
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host: "httpbin.test.svc.cluster.local",
					},
				},
			},
			expResp:   nil,
			expErrStr: "UpstreamTrafficSetting test/httpbin conflicts with test/httpbin1 since they have the same host httpbin.test.svc.cluster.local",
		},
		{
			name: "success: UpstreamTrafficSetting with duplicate host on update",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local"
						}
					}
					`),
				},
			},
			existingUpstreamTrafficSettings: []*policyv1alpha1.UpstreamTrafficSetting{
				{
					TypeMeta: metav1.TypeMeta{
						Kind:       "UpstreamTrafficSetting",
						APIVersion: "policy.openservicemesh.io/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "httpbin",
						Namespace: testNs,
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host:               "httpbin.test.svc.cluster.local",
						ConnectionSettings: nil,
					},
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "UpstreamTrafficSetting with valid rate limiting config",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local",
							"rateLimit": {
								"local": {
									"http": {
										"responseStatusCode": 429
									}
								}
							},
							"httpRoutes": [
								{
								"rateLimit": {
									"local": {
										"responseStatusCode": 503
									}
								}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "UpstreamTrafficSetting with invalid vhost rate limiting HTTP status code",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local",
							"rateLimit": {
								"local": {
									"http": {
										"responseStatusCode": 1
									}
								}
							}
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Invalid responseStatusCode 1. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
		},
		{
			name: "UpstreamTrafficSetting with invalid HTTP route rate limiting status code",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "policy.openservicemesh.io",
					Kind:    "UpstreamTrafficSetting",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "policy.openservicemesh.io/v1alpha1",
						"kind": "UpstreamTrafficSetting",
						"metadata": {
							"name": "httpbin",
							"namespace": "test"
						},
						"spec": {
							"host": "httpbin.test.svc.cluster.local",
							"rateLimit": {
								"local": {
									"http": {
										"responseStatusCode": 429
									}
								}
							},
							"httpRoutes": [
								{
								"rateLimit": {
									"local": {
										"responseStatusCode": 1
									}
								}
								}
							]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Invalid responseStatusCode 1. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			stop := make(chan struct{})
			defer close(stop)
			broker := messaging.NewBroker(stop)

			objects := make([]runtime.Object, len(tc.existingUpstreamTrafficSettings))
			for i := range tc.existingUpstreamTrafficSettings {
				objects[i] = tc.existingUpstreamTrafficSettings[i]
			}

			fakeClient := fakePolicyClientset.NewSimpleClientset(objects...)
			informerCollection, err := informers.NewInformerCollection("osm", stop,
				informers.WithPolicyClient(fakeClient),
				informers.WithKubeClient(testclient.NewSimpleClientset()),
			)
			assert.NoError(err)

			k8sClient := k8s.NewClient("test-namespace", "test-mesh-config", informerCollection, fakeClient, broker)
			policyClient := kube.NewClient(k8sClient)

			pv := &policyValidator{
				policyClient: policyClient,
			}

			if len(objects) > 0 {
				// monitor namespaces
				err := informerCollection.Add(informers.InformerKeyNamespace, newNsK8sObj(testNs), t)
				assert.Nil(err)
				assert.True(k8sClient.IsMonitoredNamespace(testNs))
			}

			// Block until we start getting upstreamtrafficsetting updates
			// We only do this because the informerCollection doesn't have the
			// policy client's msgBroker eventhandler registered when it initially runs
			// and that leads to a race condition in tests (due to the kubeController mockss)
			if len(objects) > 0 {
				ticker := time.NewTicker(1 * time.Second)
				for {
					<-ticker.C
					list, err := fakeClient.PolicyV1alpha1().UpstreamTrafficSettings(testNs).List(context.Background(), metav1.ListOptions{})
					if err != nil {
						t.Logf("Error getting UpstreamTrafficSettings: %v", err)
						continue
					}
					if len(list.Items) > 0 {
						break
					}
				}
			}

			resp, err := pv.upstreamTrafficSettingValidator(tc.input)
			assert.Equal(tc.expResp, resp)
			if tc.expErrStr == "" {
				// we expect a nil error
				assert.Nil(err)
			}
			if err != nil {
				assert.Equal(tc.expErrStr, err.Error())
			}
		})
	}
}

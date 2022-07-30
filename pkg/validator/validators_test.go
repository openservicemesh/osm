package validator

import (
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	fakePolicyClientset "github.com/openservicemesh/osm/pkg/gen/client/policy/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/policy"
)

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
			expErrStr: "expected 'port.protocol' to be 'http' or 'https', got: invalid",
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
			expErrStr: "invalid 'source.name' value specified for IPRange. Expected CIDR notation 'a.b.c.d/x', got 'invalid'",
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
			expErrStr: "invalid 'source.kind' value specified. Must be one of: Service, AuthenticatedPrincipal, IPRange",
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
			expErrStr: "invalid 'source.kind' value specified. Must be one of: Service, AuthenticatedPrincipal, IPRange",
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
			expErrStr: "duplicate backends detected with service name: test and port: 80",
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
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			stop := make(chan struct{})
			defer close(stop)
			broker := messaging.NewBroker(stop)

			objects := make([]runtime.Object, len(tc.existingIngressBackends))
			for i := range tc.existingIngressBackends {
				objects[i] = tc.existingIngressBackends[i]
			}

			// TODO: Get rid of this (it's only used for namespace monitor verification)
			k8sController := k8s.NewMockController(mockCtrl)
			if len(objects) > 0 {
				k8sController.EXPECT().IsMonitoredNamespace(gomock.Any()).Return(true)
			}

			fakeClient := fakePolicyClientset.NewSimpleClientset(objects...)
			informerCollection, err := informers.NewInformerCollection("osm", stop, informers.WithPolicyClient(fakeClient))
			assert.NoError(err)

			policyClient := policy.NewPolicyController(informerCollection, k8sController, broker)
			pv := &policyValidator{
				policyClient: policyClient,
			}

			// Block until we start getting ingressbackend updates
			// We only do this because the informerCollection doesn't have the
			// policy client's msgBroker eventhandler registered when it initially runs
			// and that leads to a race condition in tests
			if len(objects) > 0 {
				events, unsub := broker.SubscribeKubeEvents(events.IngressBackend.Added())
				<-events
				unsub()
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
			expErrStr: "expected 'matches.apiGroup' to be one of [specs.smi-spec.io/v1alpha4 policy.openservicemesh.io/v1alpha1], got: invalid",
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
			expErrStr: "cannot have more than 1 UpstreamTrafficSetting match",
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
			expErrStr: "the traffic target namespace (another-namespace) must match spec.Destination.Namespace (destination-namespace)",
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
						Namespace: "test",
					},
					Spec: policyv1alpha1.UpstreamTrafficSettingSpec{
						Host: "httpbin.test.svc.cluster.local",
					},
				},
			},
			expResp:   nil,
			expErrStr: "upstreamTrafficSetting test/httpbin conflicts with test/httpbin1 since they have the same host httpbin.test.svc.cluster.local",
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
						Namespace: "test",
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
			expErrStr: "invalid responseStatusCode 1. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
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
			expErrStr: "invalid responseStatusCode 1. See https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/v3/http_status.proto#enum-type-v3-statuscode for allowed values",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			stop := make(chan struct{})
			defer close(stop)
			broker := messaging.NewBroker(stop)

			objects := make([]runtime.Object, len(tc.existingUpstreamTrafficSettings))
			for i := range tc.existingUpstreamTrafficSettings {
				objects[i] = tc.existingUpstreamTrafficSettings[i]
			}

			// TODO: Get rid of this (it's only used for namespace monitor verification)
			k8sController := k8s.NewMockController(mockCtrl)
			if len(objects) > 0 {
				k8sController.EXPECT().IsMonitoredNamespace(gomock.Any()).Return(true)
			}

			fakeClient := fakePolicyClientset.NewSimpleClientset(objects...)
			informerCollection, err := informers.NewInformerCollection("osm", stop, informers.WithPolicyClient(fakeClient))
			assert.NoError(err)

			policyClient := policy.NewPolicyController(informerCollection, k8sController, broker)

			pv := &policyValidator{
				policyClient: policyClient,
			}

			// Block until we start getting upstreamtrafficsetting updates
			// We only do this because the informerCollection doesn't have the
			// policy client's msgBroker eventhandler registered when it initially runs
			// and that leads to a race condition in tests (due to the kubeController mockss)
			if len(objects) > 0 {
				events, unsub := broker.SubscribeKubeEvents(events.UpstreamTrafficSetting.Added())
				<-events
				unsub()
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

func TestMeshRootCertificateValidator(t *testing.T) {
	testCases := []struct {
		name         string
		input        *admissionv1.AdmissionRequest
		expErrStr    string
		existingMRCs int
	}{
		{
			name: "MeshRootCertificate with invalid Tresor certificate provider update",
			input: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Kind: metav1.GroupVersionKind{
					Group:   "configv1alpha2",
					Version: "config.openservicemesh.io",
					Kind:    "MeshRootCertificate",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"provider": {
								"tresor": {
									"ca": {
										"secretRef": {
											"name": "osm-ca-bundle",
											"namespace": "osm-system"
										}
									}
								}
							}
						}
					}
					`),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"provider": {
								"tresor": {
									"ca": {
										"secretRef": {
											"name": "new-osm-ca-bundle",
											"namespace": "test-namespace"
										}
									}
								}
							}
						}
					}
					`),
				},
			},
			expErrStr: "cannot update certificate provider settings for MRC osm-system/osm-mesh-root-certificate. Create a new MRC and initiate root certificate rotation to update the provider",
		},
		{
			name: "MeshRootCertificate with invalid trust domain update",
			input: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Update,
				Kind: metav1.GroupVersionKind{
					Group:   "configv1alpha2",
					Version: "config.openservicemesh.io",
					Kind:    "MeshRootCertificate",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"trustDomain": "newtrustdomain",
							"provider": {
								"tresor": {
							 		"ca": {
										"secretRef": {
											"name": "osm-ca-bundle",
											"namespace": "osm-system"
							  			}
							 		}
								}
							}
						}
					}
					`),
				},
				OldObject: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"trustDomain": "oldtrustdomain",
							"provider": {
								"tresor": {
							 		"ca": {
										"secretRef": {
											"name": "osm-ca-bundle",
											"namespace": "osm-system"
							  			}
							 		}
								}
							}
						}
					}
					`),
				},
			},
			expErrStr: "cannot update trust domain for MRC osm-system/osm-mesh-root-certificate. Create a new MRC and initiate root certificate rotation to update the trust domain",
		},
		{
			name: "MeshRootCertificate with invalid trust domain on create",
			input: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Kind: metav1.GroupVersionKind{
					Group:   "configv1alpha2",
					Version: "config.openservicemesh.io",
					Kind:    "MeshRootCertificate",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"trustDomain": "",
							"provider": {
								"tresor": {
							 		"ca": {
										"secretRef": {
											"name": "osm-ca-bundle",
											"namespace": "osm-system"
							  			}
							 		}
								}
							}
						}
					}
					`),
				},
			},
			expErrStr: "trustDomain must be non empty for MRC osm-system/osm-mesh-root-certificate",
		},
		{
			name: "MeshRootCertificate with no state on create",
			input: &admissionv1.AdmissionRequest{
				Operation: admissionv1.Create,
				Kind: metav1.GroupVersionKind{
					Group:   "configv1alpha2",
					Version: "config.openservicemesh.io",
					Kind:    "MeshRootCertificate",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "config.openservicemesh.io/configv1alpha2",
						"kind": "MeshRootCertificate",
						"metadata": {
							"name": "osm-mesh-root-certificate",
							"namespace": "osm-system"
						},
						"spec": {
							"trustDomain": "trustDomain",
							"provider": {
								"tresor": {
							 		"ca": {
										"secretRef": {
											"name": "osm-ca-bundle",
											"namespace": "osm-system"
							  			}
							 		}
								}
							}
						}
					}
					`),
				},
			},
			expErrStr: "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			cv := configValidator{configClient: fakeConfigClientset.NewSimpleClientset(), osmNamespace: "osm-system"}

			resp, err := cv.meshRootCertificateValidator(tc.input)
			if tc.expErrStr == "" {
				assert.NoError(err)
				assert.Nil(resp)
			} else {
				assert.Equal(tc.expErrStr, err.Error())
				assert.Nil(resp)
			}
		})
	}
}

func TestActiveRootCertificateRotation(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name         string
		mrc          configv1alpha2.MeshRootCertificate
		configClient configClientset.Interface
		expResp      bool
	}{
		{
			name: "active rotation, 2 existing MRCs",
			mrc: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "newmrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc1",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollback,
						},
					},
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollout,
						},
					},
				}...,
			),
			expResp: true,
		},
		{
			name: "no active rotation, 2 existing MRCs",
			mrc: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "newmrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc1",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateInactive,
						},
					},
				}...,
			),
			expResp: false,
		},
		{
			name: "no active rotation, 2 existing MRCs, 1 in error state",
			mrc: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "newmrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc1",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateError,
						},
					},
				}...,
			),
			expResp: false,
		},
		{
			name: "no active rotation, 2 existing MRCs, one of which is the mrc being updated",
			mrc: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc2",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateIssuingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc1",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
					&configv1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: configv1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateValidatingRollout,
						},
					},
				}...,
			),
			expResp: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cv := configValidator{configClient: tc.configClient, osmNamespace: "osm-namespace"}

			active := cv.activeRootCertificateRotation(&tc.mrc)

			assert.Equal(tc.expResp, active)
		})
	}
}

func TestValidateMeshRootCertificateStatusTransition(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name   string
		oldMRC configv1alpha2.MeshRootCertificate
		newMRC configv1alpha2.MeshRootCertificate
		expErr bool
	}{
		{
			name: "no status change",
			oldMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			newMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			expErr: false,
		},
		{
			name: "valid status transition from '' to 'validatingRollout'",
			oldMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
			},
			newMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			expErr: false,
		},
		{
			name: "unknown old mrc state 'running'",
			oldMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: "running",
				},
			},
			newMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateIssuingRollback,
				},
			},
			expErr: true,
		},
		{
			name: "unknown new mrc state 'rollback'",
			oldMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateIssuingRollback,
				},
			},
			newMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: "rollback",
				},
			},
			expErr: true,
		},
		{
			name: "invalid status update from 'validatingRollout' to 'active'",
			oldMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			newMRC: configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: configv1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			},
			expErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMeshRootCertificateStatusTransition(&tc.oldMRC, &tc.newMRC)

			if tc.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

func TestValidateMeshRootCertificateStatusCombination(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name         string
		mrc          v1alpha2.MeshRootCertificate
		configClient configClientset.Interface
		expErr       bool
	}{
		{
			name: "invalid status combination, validatingRollback and issuingRollout",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollout,
						},
					},
				}...,
			),
			expErr: true,
		},
		{
			name: "no active rotation, update from no status to active",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "no active rotation, update from active to error",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateError,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, validatingRollout and active",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, issuingRollout and active",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateIssuingRollout,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateValidatingRollout,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, active and issuingRollback",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollout,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollback,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, issuingRollback and issuingRollout",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateIssuingRollback,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollout,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, validatingRollback and active",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateValidatingRollback,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollback,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, inactive and active",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateInactive,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateValidatingRollback,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
		{
			name: "valid status combination, error state",
			mrc: v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mrc",
					Namespace: "osm-namespace",
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateError,
				},
			},
			configClient: fakeConfigClientset.NewSimpleClientset(
				[]runtime.Object{
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateIssuingRollback,
						},
					},
					&v1alpha2.MeshRootCertificate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "mrc2",
							Namespace: "osm-namespace",
						},
						Status: v1alpha2.MeshRootCertificateStatus{
							State: constants.MRCStateActive,
						},
					},
				}...,
			),
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cv := configValidator{configClient: tc.configClient, osmNamespace: "osm-namespace"}

			err := cv.validateMeshRootCertificateStatusCombination(&tc.mrc)

			if tc.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}

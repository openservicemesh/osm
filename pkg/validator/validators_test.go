package validator

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestIngressBackendValidator(t *testing.T) {
	testCases := []struct {
		name      string
		input     *admissionv1.AdmissionRequest
		expResp   *admissionv1.AdmissionResponse
		expErrStr string
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			resp, err := ingressBackendValidator(tc.input)
			assert.Equal(tc.expResp, resp)
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

func TestMulticlusterServiceValidator(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name      string
		input     *admissionv1.AdmissionRequest
		expResp   *admissionv1.AdmissionResponse
		expErrStr string
	}{
		{
			name: "MultiClusterService with empty name fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"serviceAccount" : "sdf",
							"clusters": [{
								"name": "",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Cluster name is not valid",
		},
		{
			name: "MultiClusterService with duplicate cluster names fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"clusters": [{
								"name": "test",
								"address": "0.0.0.0:8080"
							},{
								"name": "test",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Cluster named test already exists",
		},
		{
			name: "MultiClusterService has an acceptable name",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"clusters": [{
								"name": "test",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "",
		},
		{
			name: "MultiClusterService with empty address fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"serviceAccount" : "sdf",
							"clusters": [{
								"name": "test",
								"address": ""
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Cluster address  is not valid",
		},
		{
			name: "MultiClusterService with invalid IP fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"clusters": [{
								"name": "test",
								"address": "0.0.00:22"
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Error parsing IP address 0.0.00:22",
		},
		{
			name: "MultiClusterService with invalid port fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MultiClusterService",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "MultiClusterService",
						"spec": {
							"clusters": [{
								"name": "test",
								"address": "0.0.0.0:a"
							}]
						}
					}
					`),
				},
			},
			expResp:   nil,
			expErrStr: "Error parsing port value 0.0.0.0:a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := MultiClusterServiceValidator(tc.input)
			t.Log(tc.input.Kind.Kind)
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

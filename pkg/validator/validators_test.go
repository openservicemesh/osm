package validator

import (
	"errors"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEgressValidator(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name    string
		input   *admissionv1.AdmissionRequest
		expResp *admissionv1.AdmissionResponse
		expErr  error
	}{
		{
			name: "Egress with bad http route fails",
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
								"apiGroup": "v1alpha1",
								"kind": "BadHttpRoute",
								"name": "Name"
								}
							]
						}
					}
					`),
				},
			},

			expResp: nil,
			expErr:  errors.New("Expected Matches.Kind to be 'HTTPRouteGroup', got: BadHttpRoute"),
		},
		{
			name: "Egress with bad API group fails",
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
								"apiGroup": "test",
								"kind": "HTTPRouteGroup",
								"name": "Name"
								}
							]
						}
					}
					`),
				},
			},

			expResp: nil,
			expErr:  errors.New("Expected Matches.APIGroup to be 'specs.smi-spec.io/v1alpha4', got: test"),
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

			expResp: nil,
			expErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := EgressValidator(tc.input)
			t.Log(tc.input.Kind.Kind)
			assert.Equal(tc.expResp, resp)
			assert.Equal(tc.expErr, err)
		})
	}
}

func TestMeshConfigValidator(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name    string
		input   *admissionv1.AdmissionRequest
		expResp *admissionv1.AdmissionResponse
		expErr  error
	}{
		{
			name: "MeshConfig with invalid duration fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MeshConfig",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"certificate": {
								"serviceCertValidityDuration": "abc"
							}
						}
					}
					`),
				},
			},

			expResp: nil,
			expErr:  errors.New("Certificate.ServiceCertValidityDuration abc is not valid"),
		},
		{
			name: "MeshConfig with duration lower than minimum duration fails",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MeshConfig",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"certificate": {
								"serviceCertValidityDuration": "0.5s"
							}
						}
					}
					`),
				},
			},

			expResp: nil,
			expErr:  errors.New("Certificate.ServiceCertValidityDuration 500000000 is lower than 1000000000"),
		},
		{
			name: "MeshConfig with valid duration passes",
			input: &admissionv1.AdmissionRequest{
				Kind: metav1.GroupVersionKind{
					Group:   "v1alpha1",
					Version: "config.openservicemesh.io",
					Kind:    "MeshConfig",
				},
				Object: runtime.RawExtension{
					Raw: []byte(`
					{
						"apiVersion": "v1alpha1",
						"kind": "Egress",
						"spec": {
							"certificate": {
								"serviceCertValidityDuration": "1h"
							}
						}
					}
					`),
				},
			},

			expResp: nil,
			expErr:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := MeshConfigValidator(tc.input)
			t.Log(tc.input.Kind.Kind)
			assert.Equal(tc.expResp, resp)
			assert.Equal(tc.expErr, err)
		})
	}
}

func TestMulticlusterServiceValidator(t *testing.T) {
	assert := tassert.New(t)
	testCases := []struct {
		name    string
		input   *admissionv1.AdmissionRequest
		expResp *admissionv1.AdmissionResponse
		expErr  error
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
							"cluster": [{
								"name": "",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  errors.New("Cluster name  is not valid"),
		},
		{
			name: "MultiClusterService with global name fails",
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
							"cluster": [{
								"name": "global",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  errors.New("Cluster name global is not valid"),
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
							"cluster": [{
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
			expResp: nil,
			expErr:  errors.New("Cluster named test already exists"),
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
							"cluster": [{
								"name": "test",
								"address": "0.0.0.0:8080"
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  nil,
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
							"cluster": [{
								"name": "test",
								"address": ""
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  errors.New("Cluster address  is not valid"),
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
							"cluster": [{
								"name": "test",
								"address": "0.0.00:22"
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  errors.New("Error parsing IP address 0.0.00:22"),
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
							"cluster": [{
								"name": "test",
								"address": "0.0.0.0:a"
							}]
						}
					}
					`),
				},
			},
			expResp: nil,
			expErr:  errors.New("Error parsing port value 0.0.0.0:a"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := MultiClusterServiceValidator(tc.input)
			t.Log(tc.input.Kind.Kind)
			assert.Equal(tc.expResp, resp)
			assert.Equal(tc.expErr, err)
		})
	}
}

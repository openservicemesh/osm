package crdconversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	configv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
)

func TestConvertMeshConfig(t *testing.T) {
	testCases := []struct {
		name      string
		request   runtime.Object
		toVersion string
		verifyFn  func(*assert.Assertions, *unstructured.Unstructured, error)
	}{
		{
			name: "v1alpha3 -> v1alpha2 should remove additional field",
			request: &configv1alpha3.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha3",
					Kind:       "MeshConfig",
				},
				Spec: configv1alpha3.MeshConfigSpec{
					Traffic: configv1alpha3.TrafficSpec{
						OutboundIPRangeInclusionList: []string{"1.1.1.1/32"},
					},
					Sidecar: configv1alpha3.SidecarSpec{
						TLSMinProtocolVersion: "TLSv1_2",
						TLSMaxProtocolVersion: "TLSv1_3",
						CipherSuites:          []string{"SomeCipherSuite"},
						ECDHCurves:            []string{"SomeECDHCurve"},
						LocalProxyMode:        configv1alpha3.LocalProxyModePodIP,
					},
				},
			},
			toVersion: "config.openservicemesh.io/v1alpha2",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)

				unsupportedFields := [][]string{
					{"spec", "sidecar", "localProxyMode"},
				}

				for _, unsupportedField := range unsupportedFields {
					_, found, _ := unstructured.NestedSlice(converted.Object, unsupportedField...)
					a.False(found)
				}
			},
		},
		{
			name: "v1alpha3 -> v1alpha1 should remove additional field",
			request: &configv1alpha3.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha3",
					Kind:       "MeshConfig",
						OutboundIPRangeInclusionList: []string{"1.1.1.1/32"},
					},
					Sidecar: configv1alpha3.SidecarSpec{
						TLSMinProtocolVersion: "TLSv1_2",
						TLSMaxProtocolVersion: "TLSv1_3",
						CipherSuites:          []string{"SomeCipherSuite"},
						ECDHCurves:            []string{"SomeECDHCurve"},
						LocalProxyMode:        configv1alpha3.LocalProxyModePodIP,
					},
				},
			},
			toVersion: "config.openservicemesh.io/v1alpha1",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)

				unsupportedFields := [][]string{
					{"spec", "traffic", "outboundIPRangeInclusionList"},
					{"spec", "sidecar", "tlsMinProtocolVersion"},
					{"spec", "sidecar", "tlsMaxProtocolVersion"},
					{"spec", "sidecar", "cipherSuites"},
					{"spec", "sidecar", "ecdhCurves"},
					{"spec", "sidecar", "localProxyMode"},
				}

				for _, unsupportedField := range unsupportedFields {
					_, found, _ := unstructured.NestedSlice(converted.Object, unsupportedField...)
					a.False(found)
				}
			},
		},
		{
			name: "v1alpha2 -> v1alpha1 should remove additional field",
			request: &configv1alpha2.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha2",
					Kind:       "MeshConfig",
				},
				Spec: configv1alpha2.MeshConfigSpec{
					Traffic: configv1alpha2.TrafficSpec{
						OutboundIPRangeInclusionList: []string{"1.1.1.1/32"},
					},
					Sidecar: configv1alpha2.SidecarSpec{
						TLSMinProtocolVersion: "TLSv1_2",
						TLSMaxProtocolVersion: "TLSv1_3",
						CipherSuites:          []string{"SomeCipherSuite"},
						ECDHCurves:            []string{"SomeECDHCurve"},
					},
				},
			},
			toVersion: "config.openservicemesh.io/v1alpha1",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)

				unsupportedFields := [][]string{
					{"spec", "traffic", "outboundIPRangeInclusionList"},
					{"spec", "sidecar", "tlsMinProtocolVersion"},
					{"spec", "sidecar", "tlsMaxProtocolVersion"},
					{"spec", "sidecar", "cipherSuites"},
					{"spec", "sidecar", "ecdhCurves"},
				}

				for _, unsupportedField := range unsupportedFields {
					_, found, _ := unstructured.NestedSlice(converted.Object, unsupportedField...)
					a.False(found)
				}
			},
		},
		{
			name: "v1alpha1 -> v1alpha2",
			request: &configv1alpha1.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha1",
					Kind:       "MeshConfig",
				},
				Spec: configv1alpha1.MeshConfigSpec{},
			},
			toVersion: "config.openservicemesh.io/v1alpha2",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)
			},
		},
		{
			name: "v1alpha1 -> v1alpha3",
			request: &configv1alpha1.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha1",
					Kind:       "MeshConfig",
				},
				Spec: configv1alpha1.MeshConfigSpec{},
			},
			toVersion: "config.openservicemesh.io/v1alpha3",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)
			},
		},
		{
			name: "v1alpha2 -> v1alpha3",
			request: &configv1alpha2.MeshConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openservicemesh.io/v1alpha2",
					Kind:       "MeshConfig",
				},
				Spec: configv1alpha2.MeshConfigSpec{},
			},
			toVersion: "config.openservicemesh.io/v1alpha3",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, err error) {
				a.NoError(err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tc.request)
			a.Nil(err)
			unstructuredReq := &unstructured.Unstructured{Object: obj}
			converted, status := convertMeshConfig(unstructuredReq, tc.toVersion)
			tc.verifyFn(a, converted, status)
		})
	}
}

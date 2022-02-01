package crdconversion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	configv1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

func TestConvertMeshConfig(t *testing.T) {
	testCases := []struct {
		name      string
		request   runtime.Object
		toVersion string
		verifyFn  func(*assert.Assertions, *unstructured.Unstructured, metav1.Status)
	}{
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
				},
			},
			toVersion: "config.openservicemesh.io/v1alpha1",
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, status metav1.Status) {
				a.Equal(status, statusSucceed())

				_, found, _ := unstructured.NestedSlice(converted.Object, "spec", "traffic", "outboundIPRangeInclusionList")
				a.False(found)
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
			verifyFn: func(a *assert.Assertions, converted *unstructured.Unstructured, status metav1.Status) {
				a.Equal(status, statusSucceed())
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

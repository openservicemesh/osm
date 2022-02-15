package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestSingleMeshOK(t *testing.T) {
	tests := []struct {
		name              string
		enforceSingleMesh bool
		resources         []runtime.Object
		expectedOK        bool
	}{
		{
			name:              "no existing resources, single mesh enforced",
			enforceSingleMesh: true,
			expectedOK:        true,
		},
		{
			name:              "no existing resources, single mesh not enforced",
			enforceSingleMesh: false,
			expectedOK:        true,
		},
		{
			name:              "existing mesh not enforcing single mesh, new mesh enforcing single mesh",
			enforceSingleMesh: true,
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "any-namespace",
						Labels: map[string]string{
							constants.AppLabel: constants.OSMControllerName,
						},
					},
				},
			},
			expectedOK: false,
		},
		{
			name:              "existing mesh enforcing single mesh, new mesh not enforcing single mesh",
			enforceSingleMesh: false,
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "any-namespace",
						Labels: map[string]string{
							constants.AppLabel:  constants.OSMControllerName,
							"enforceSingleMesh": "true",
						},
					},
				},
			},
			expectedOK: false,
		},
		{
			name:              "existing mesh not enforcing single mesh, new mesh not enforcing single mesh",
			enforceSingleMesh: false,
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "any-namespace",
						Labels: map[string]string{
							constants.AppLabel: constants.OSMControllerName,
						},
					},
				},
			},
			expectedOK: true,
		},
		{
			name:              "existing mesh enforcing single mesh, new mesh enforcing single mesh",
			enforceSingleMesh: true,
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "any-namespace",
						Labels: map[string]string{
							constants.AppLabel:  constants.OSMControllerName,
							"enforceSingleMesh": "true",
						},
					},
				},
			},
			expectedOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := singleMeshOK(fake.NewSimpleClientset(test.resources...), test.enforceSingleMesh)()
			if test.expectedOK {
				assert.NoError(t, err)
			} else {
				t.Log(err)
				assert.Error(t, err)
			}
		})
	}
}

func TestNamespaceHasNoMesh(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		resources  []runtime.Object
		expectedOK bool
	}{
		{
			name:       "no existing resources",
			namespace:  "osm-system",
			expectedOK: true,
		},
		{
			name:      "existing controller outside namespace",
			namespace: "osm-system",
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "not-osm-system",
						Labels: map[string]string{
							constants.AppLabel: constants.OSMControllerName,
						},
					},
				},
			},
			expectedOK: true,
		},
		{
			name:      "existing controller in namespace",
			namespace: "osm-system",
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "osm-system",
						Labels: map[string]string{
							constants.AppLabel: constants.OSMControllerName,
						},
					},
				},
			},
			expectedOK: false,
		},
		{
			name:      "existing non-controller in namespace",
			namespace: "osm-system",
			resources: []runtime.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "osm-system",
						Labels: map[string]string{
							constants.AppLabel: "not-" + constants.OSMControllerName,
						},
					},
				},
			},
			expectedOK: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := namespaceHasNoMesh(fake.NewSimpleClientset(test.resources...), test.namespace)()
			if test.expectedOK {
				assert.NoError(t, err)
			} else {
				t.Log(err)
				assert.Error(t, err)
			}
		})
	}
}

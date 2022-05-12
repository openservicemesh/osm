package reconciler

import (
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiservertestclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

const (
	meshName   = "test-mesh"
	osmVersion = "test-version"
)

func TestCRDEventHandlerUpdateFunc(t *testing.T) {
	testCases := []struct {
		name        string
		originalCrd apiv1.CustomResourceDefinition
		updatedCrd  apiv1.CustomResourceDefinition
		crdUpdated  bool
	}{
		{
			name: "crd spec changed",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  false,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: true,
		},
		{
			name: "crd new label added",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
						"some":                       "label",
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: false,
		},
		{
			name: "crd name changed",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io.NEW",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := tassert.New(t)

			kubeClient := testclient.NewSimpleClientset()
			apiServerClient := apiservertestclient.NewSimpleClientset(&tc.originalCrd)

			c := client{
				kubeClient:      kubeClient,
				meshName:        meshName,
				apiServerClient: apiServerClient,
				informers:       informerCollection{},
			}
			// Invoke update handler
			handlers := c.crdEventHandler()
			handlers.UpdateFunc(&tc.originalCrd, &tc.updatedCrd)

			if tc.crdUpdated {
				a.Equal(&tc.originalCrd, &tc.updatedCrd)
			} else {
				a.NotEqual(&tc.originalCrd, &tc.updatedCrd)
			}

			crd, err := c.apiServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), tc.originalCrd.Name, metav1.GetOptions{})
			a.Nil(err)

			if tc.crdUpdated {
				a.Equal(crd, &tc.updatedCrd)
			} else {
				a.Equal(crd, &tc.originalCrd)
			}
		})
	}
}

func TestCRDEventHandlerDeleteFunc(t *testing.T) {
	originalCrd := apiv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "meshconfigs.config.openservicemesh.io",
			Labels: map[string]string{
				constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
				constants.ReconcileLabel:     strconv.FormatBool(true),
			},
		},
		Spec: apiv1.CustomResourceDefinitionSpec{
			Group: "config.openservicemesh.io",
			Names: apiv1.CustomResourceDefinitionNames{
				Plural:     "meshconfigs",
				Singular:   "meshconfig",
				ShortNames: []string{"meshconfig"},
				Kind:       "MeshConfig",
				ListKind:   "MeshConfigList",
			},
			Scope: "Namespaced",
			Versions: []apiv1.CustomResourceDefinitionVersion{{
				Name:    "v1alpha1",
				Served:  true,
				Storage: true,
				Schema: &apiv1.CustomResourceValidation{
					OpenAPIV3Schema: &apiv1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]apiv1.JSONSchemaProps{
							"spec": {
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"sidecar": {
										Type:        "object",
										Description: "Configuration for Envoy sidecar",
										Properties: map[string]apiv1.JSONSchemaProps{
											"enablePrivilegedInitContainer": {
												Type:        "boolean",
												Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
											},
											"logLevel": {
												Type:        "string",
												Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
											},
										},
									},
								},
							},
						},
					},
				},
			}},
		},
	}

	a := tassert.New(t)
	kubeClient := testclient.NewSimpleClientset()
	apiServerClient := apiservertestclient.NewSimpleClientset()

	c := client{
		kubeClient:      kubeClient,
		meshName:        meshName,
		apiServerClient: apiServerClient,
		informers:       informerCollection{},
	}
	// Invoke delete handler
	handlers := c.crdEventHandler()
	handlers.DeleteFunc(&originalCrd)

	// verify crd exists after deletion
	crd, err := c.apiServerClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), originalCrd.Name, metav1.GetOptions{})
	a.Nil(err)
	a.Equal(crd, &originalCrd)
}

func TestIsCRDUpdated(t *testing.T) {
	testCases := []struct {
		name        string
		originalCrd apiv1.CustomResourceDefinition
		updatedCrd  apiv1.CustomResourceDefinition
		crdUpdated  bool
	}{
		{
			name: "crd spec changed",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  false,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: true,
		},
		{
			name: "crd new label added",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
						"some":                       "label",
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: false,
		},
		{
			name: "crd name changed",
			originalCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			updatedCrd: apiv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "meshconfigs.config.openservicemesh.io.NEW",
					Labels: map[string]string{
						constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
						constants.ReconcileLabel:     strconv.FormatBool(true),
					},
				},
				Spec: apiv1.CustomResourceDefinitionSpec{
					Group: "config.openservicemesh.io",
					Names: apiv1.CustomResourceDefinitionNames{
						Plural:     "meshconfigs",
						Singular:   "meshconfig",
						ShortNames: []string{"meshconfig"},
						Kind:       "MeshConfig",
						ListKind:   "MeshConfigList",
					},
					Scope: "Namespaced",
					Versions: []apiv1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
						Schema: &apiv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiv1.JSONSchemaProps{
									"spec": {
										Type: "object",
										Properties: map[string]apiv1.JSONSchemaProps{
											"sidecar": {
												Type:        "object",
												Description: "Configuration for Envoy sidecar",
												Properties: map[string]apiv1.JSONSchemaProps{
													"enablePrivilegedInitContainer": {
														Type:        "boolean",
														Description: "Enables privileged init containers for pods in mesh. When false, init containers only have NET_ADMIN.",
													},
													"logLevel": {
														Type:        "string",
														Description: "Sets the logging verbosity of Envoy proxy sidecar, only applicable to newly created pods joining the mesh.",
													},
												},
											},
										},
									},
								},
							},
						},
					}},
				},
			},
			crdUpdated: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			result := isCRDUpdated(&tc.originalCrd, &tc.updatedCrd)
			assert.Equal(result, tc.crdUpdated)
		})
	}
}

func TestIsLabelModified(t *testing.T) {
	testCases := []struct {
		name          string
		labelMap      map[string]string
		key           string
		value         string
		labelModified bool
	}{
		{
			name: "labels not modified",
			labelMap: map[string]string{
				constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
				constants.ReconcileLabel:     strconv.FormatBool(true),
			},
			key:           constants.OSMAppNameLabelKey,
			value:         constants.OSMAppNameLabelValue,
			labelModified: false,
		},
		{
			name: "labels modified",
			labelMap: map[string]string{
				constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
			},
			key:           constants.OSMAppNameLabelKey,
			value:         "test",
			labelModified: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			result := isLabelModified(tc.key, tc.value, tc.labelMap)
			assert.Equal(result, tc.labelModified)
		})
	}
}

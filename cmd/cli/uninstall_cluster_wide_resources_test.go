package main

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsClientSetFake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/injector"
	"github.com/openservicemesh/osm/pkg/validator"
)

var (
	someOtherNamespace                = "someOtherNamespace"
	someOtherCustomResourceDefinition = "someOtherCustomResourceDefinition"
	someOtherWebhookName              = "someOtherWebhookName"
	someOtherSecretName               = "someOtherSecretName"
)

func TestUninstallClusterWideResources(t *testing.T) {
	tests := []struct {
		name                                    string
		existingNamespaces                      []runtime.Object
		existingCustomResourceDefinitions       []runtime.Object
		existingMutatingWebhookConfigurations   []runtime.Object
		existingValidatingWebhookConfigurations []runtime.Object
		existingSecrets                         []runtime.Object
	}{
		{
			name: "osm/smi resources EXIST before run, should NOT ERROR, osm/smi resources should BE DELETED, non-osm/non-smi resources should NOT BE DELETED",
			existingNamespaces: []runtime.Object{
				// OSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNamespace,
					},
				},
				// non-OSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherNamespace,
					},
				},
			},
			existingCustomResourceDefinitions: []runtime.Object{
				// OSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "egresses.policy.openservicemesh.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// OSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ingressbackends.policy.openservicemesh.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// OSM CRD
				&apiv1.CustomResourceDefinition{
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
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// OSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "multiclusterservices.config.openservicemesh.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "httproutegroups.specs.smi-spec.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "tcproutes.specs.smi-spec.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "trafficsplits.split.smi-spec.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "traffictargets.access.smi-spec.io",
						Labels: map[string]string{
							constants.OSMAppNameLabelKey: constants.OSMAppNameLabelValue,
							constants.ReconcileLabel:     strconv.FormatBool(true),
						},
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
				// non-OSM/non-SMI CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherCustomResourceDefinition,
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
			},
			existingMutatingWebhookConfigurations: []runtime.Object{
				// OSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: injector.MutatingWebhookName,
						Labels: map[string]string{
							constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
							constants.OSMAppInstanceLabelKey: testMeshName,
							constants.AppLabel:               constants.OSMInjectorName,
						},
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
				// non-OSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			},
			existingValidatingWebhookConfigurations: []runtime.Object{
				// OSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: validator.ValidatingWebhookName,
						Labels: map[string]string{
							constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
							constants.OSMAppInstanceLabelKey: testMeshName,
							constants.AppLabel:               constants.OSMControllerName,
						},
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
				// non-OSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			},
			existingSecrets: []runtime.Object{
				// OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.DefaultCABundleSecretName,
						Namespace: testNamespace,
					},
				},
				// OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.CrdConverterCertificateSecretName,
						Namespace: testNamespace,
					},
				},
				// OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.MutatingWebhookCertificateSecretName,
						Namespace: testNamespace,
					},
				},
				// OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.ValidatingWebhookCertificateSecretName,
						Namespace: testNamespace,
					},
				},
				// non-OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      someOtherSecretName,
						Namespace: testNamespace,
					},
				},
			},
		},
		{
			name: "osm/smi resources DO NOT EXIST before run, should NOT ERROR, non-osm/non-smi resources should NOT BE DELETED",
			existingNamespaces: []runtime.Object{
				// non-OSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherNamespace,
					},
				},
			},
			existingCustomResourceDefinitions: []runtime.Object{
				// non-OSM CRD
				&apiv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						Kind:       "CustomResourceDefinition",
						APIVersion: "apiextensions.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherCustomResourceDefinition,
					},
					Spec: apiv1.CustomResourceDefinitionSpec{},
				},
			},
			existingMutatingWebhookConfigurations: []runtime.Object{
				// non-OSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			},
			existingValidatingWebhookConfigurations: []runtime.Object{
				// non-OSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: someOtherWebhookName,
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			},
			existingSecrets: []runtime.Object{
				// non-OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      someOtherSecretName,
						Namespace: testNamespace,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			var existingKubeClientsetObjects []runtime.Object
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingNamespaces...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingMutatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingValidatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, test.existingSecrets...)

			uninstall := uninstallClusterWideResourcesCmd{
				in:                 in,
				out:                out,
				force:              true,
				meshName:           testMeshName,
				meshNamespace:      testNamespace,
				caBundleSecretName: constants.DefaultCABundleSecretName,
				clientset:          fake.NewSimpleClientset(existingKubeClientsetObjects...),
				// CustomResourceDefinitions belong to the extensionsClientset
				extensionsClientset: extensionsClientSetFake.NewSimpleClientset(test.existingCustomResourceDefinitions...),
			}

			err := uninstall.run()
			assert.Nil(err)

			// ensure that only the non-OSM CRDs remain in the cluster
			crdsList, err := uninstall.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(crdsList.Items))
			assert.Equal(someOtherCustomResourceDefinition, crdsList.Items[0].Name)

			// ensure that only the non-OSM MutatingWebhookConfigurations remain in the cluster
			mutatingWebhookConfigurations, err := uninstall.clientset.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(mutatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, mutatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-OSM ValidatingWebhookConfigurations remain in the cluster
			validatingWebhookConfigurations, err := uninstall.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(validatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, validatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-OSM Secrets remain in the cluster
			secrets, err := uninstall.clientset.CoreV1().Secrets(uninstall.meshNamespace).List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(secrets.Items))
			assert.Equal(someOtherSecretName, secrets.Items[0].Name)

			// ensure that existing namespaces are not deleted as deleting namespaces could be disastrous (for example, if OSM was installed in namespace kube-system)
			namespaces, err := uninstall.clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(len(test.existingNamespaces), len(namespaces.Items))
		})
	}
}

func TestUninstallClusterWideResourcesValidateCLIParams(t *testing.T) {
	tests := []struct {
		name               string
		meshName           string
		meshNamespace      string
		caBundleSecretName string
	}{
		{
			name:               "empty meshName - should error",
			meshName:           "",
			meshNamespace:      testNamespace,
			caBundleSecretName: constants.DefaultCABundleSecretName,
		},
		{
			name:               "empty meshNamespace - should error",
			meshName:           testMeshName,
			meshNamespace:      "",
			caBundleSecretName: constants.DefaultCABundleSecretName,
		},
		{
			name:               "empty caBundleSecretName - should error",
			meshName:           testMeshName,
			meshNamespace:      testNamespace,
			caBundleSecretName: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			uninstall := uninstallClusterWideResourcesCmd{
				in:                  in,
				out:                 out,
				force:               true,
				meshName:            test.meshName,
				meshNamespace:       test.meshNamespace,
				caBundleSecretName:  test.caBundleSecretName,
				clientset:           fake.NewSimpleClientset(),
				extensionsClientset: extensionsClientSetFake.NewSimpleClientset(),
			}

			err := uninstall.run()
			assert.NotNil(err)
		})
	}
}

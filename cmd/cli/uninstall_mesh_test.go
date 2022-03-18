package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
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
	osmTestVersion                    = "testVersion"
)

func TestUninstallCmd(t *testing.T) {
	tests := []struct {
		name            string
		meshName        string
		meshNamespace   string
		force           bool
		deleteNamespace bool
		meshExists      bool
		userPromptsYes  bool
	}{
		{
			name:            "the mesh DOES NOT exists",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: false,
			userPromptsYes:  true,
			meshExists:      false,
		},
		{
			name:            "the mesh DOES NOT exists and force delete set to true",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           true,
			deleteNamespace: false,
			userPromptsYes:  true,
			meshExists:      false,
		},
		{
			name:            "the mesh exists",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: false,
			userPromptsYes:  true,
			meshExists:      true,
		},
		{
			name:            "the mesh exists and force set to true",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           true,
			deleteNamespace: false,
			userPromptsYes:  true,
			meshExists:      true,
		},
		{
			name:            "the mesh exists, force set to true and delete namespace set to true",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           true,
			deleteNamespace: true,
			userPromptsYes:  true,
			meshExists:      true,
		},
		{
			name:            "the mesh exists, force set to false, delete namespace set to true and user refuses mesh delete",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: true,
			meshExists:      true,
			userPromptsYes:  true,
		},
		{
			name:            "the mesh exists, force set to false, delete namespace set to true and user approves mesh delete",
			meshName:        testMeshName,
			meshNamespace:   testNamespace,
			force:           false,
			deleteNamespace: true,
			meshExists:      true,
			userPromptsYes:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			out := new(bytes.Buffer)
			in := new(bytes.Buffer)

			if test.userPromptsYes {
				in.Write([]byte("y\n"))
			} else {
				in.Write([]byte("n\n"))
			}

			var existingKubeClientsetObjects []runtime.Object
			existingNamespaces := []runtime.Object{
				// OSM control plane namespace
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: test.meshNamespace,
					},
				},
			}
			existingCustomResourceDefinitions := []runtime.Object{
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
			}
			existingMutatingWebhookConfigurations := []runtime.Object{
				// OSM MutatingWebhookConfiguration
				&admissionregistrationv1.MutatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: injector.MutatingWebhookName,
						Labels: map[string]string{
							constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
							constants.OSMAppInstanceLabelKey: test.meshName,
							constants.AppLabel:               constants.OSMInjectorName,
						},
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{},
				},
			}
			existingValidatingWebhookConfigurations := []runtime.Object{
				// OSM ValidatingWebhookConfiguration
				&admissionregistrationv1.ValidatingWebhookConfiguration{
					ObjectMeta: metav1.ObjectMeta{
						Name: validator.ValidatingWebhookName,
						Labels: map[string]string{
							constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
							constants.OSMAppInstanceLabelKey: test.meshName,
							constants.AppLabel:               constants.OSMControllerName,
						},
					},
					Webhooks: []admissionregistrationv1.ValidatingWebhook{},
				},
			}
			existingSecrets := []runtime.Object{
				// OSM Secret
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.DefaultCABundleSecretName,
						Namespace: test.meshNamespace,
					},
				},
			}
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingNamespaces...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingMutatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingValidatingWebhookConfigurations...)
			existingKubeClientsetObjects = append(existingKubeClientsetObjects, existingSecrets...)

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(test.meshNamespace)
			}

			if test.meshExists {
				rel := release.Mock(&release.MockReleaseOptions{Name: test.meshName})
				err := store.Create(rel)
				Expect(err).To(BeNil())
			}

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet := fake.NewSimpleClientset(existingKubeClientsetObjects...)

			if test.meshExists {
				_, err := addDeployment(fakeClientSet, "osm-controller-1", test.meshName, test.meshNamespace, osmTestVersion, true)
				Expect(err).NotTo(HaveOccurred())
			}

			uninstall := uninstallMeshCmd{
				in:                  in,
				out:                 out,
				force:               test.force,
				client:              helm.NewUninstall(testConfig),
				meshName:            test.meshName,
				meshNamespace:       test.meshNamespace,
				caBundleSecretName:  constants.DefaultCABundleSecretName,
				clientSet:           fakeClientSet,
				extensionsClientset: extensionsClientSetFake.NewSimpleClientset(existingCustomResourceDefinitions...),
				deleteNamespace:     test.deleteNamespace,
			}

			err := uninstall.run()

			if !test.force {
				if test.meshExists {
					assert.Contains(out.String(), fmt.Sprintf("\nUninstall OSM [mesh name: %s] in namespace [%s] and/or OSM resources ? [y/n]: ", test.meshName, test.meshNamespace))
				} else {
					assert.Contains(out.String(), "\nList of meshes present in the cluster:\nNo osm mesh control planes found\n")
					assert.Contains(out.String(), fmt.Sprintf("\nUninstall OSM [mesh name: %s] in namespace [%s] and/or OSM resources ? [y/n]: ", test.meshName, test.meshNamespace))
				}
			}

			if test.meshExists {
				assert.Contains(out.String(), fmt.Sprintf("OSM [mesh name: %s] in namespace [%s] uninstalled\n", test.meshName, test.meshNamespace))
				assert.Nil(err)
			} else {
				assert.Contains(out.String(), fmt.Sprintf("No OSM control plane with mesh name [%s] found in namespace [%s]", test.meshName, test.meshNamespace))
			}

			if test.userPromptsYes {
				if test.deleteNamespace {
					assert.Contains(out.String(), fmt.Sprintf("OSM namespace [%s] deleted successfully\n", test.meshNamespace))
					namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
					assert.Nil(err)
					assert.Equal(0, len(namespaces.Items))
				} else {
					namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
					assert.Nil(err)
					assert.Equal(len(existingNamespaces), len(namespaces.Items))
				}
			}

			// ensure that the OSM CRDs remain in the cluster
			crdsList, err := uninstall.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(crdsList.Items))
			assert.Equal("egresses.policy.openservicemesh.io", crdsList.Items[0].Name)

			// ensure that the OSM MutatingWebhookConfigurations remain in the cluster
			mutatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(mutatingWebhookConfigurations.Items))
			assert.Equal(injector.MutatingWebhookName, mutatingWebhookConfigurations.Items[0].Name)

			// ensure that OSM ValidatingWebhookConfigurations remain in the cluster
			validatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(validatingWebhookConfigurations.Items))
			assert.Equal(validator.ValidatingWebhookName, validatingWebhookConfigurations.Items[0].Name)

			// ensure that OSM Secrets remain in the cluster
			secrets, err := uninstall.clientSet.CoreV1().Secrets(uninstall.meshNamespace).List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(len(secrets.Items), 1)
			assert.Equal(constants.DefaultCABundleSecretName, secrets.Items[0].Name)
		})
	}
}

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

			store := storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(testNamespace)
			}

			rel := release.Mock(&release.MockReleaseOptions{Name: testMeshName})
			err := store.Create(rel)
			Expect(err).To(BeNil())

			testConfig := &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			uninstall := uninstallMeshCmd{
				in:                 in,
				out:                out,
				force:              true,
				client:             helm.NewUninstall(testConfig),
				meshName:           testMeshName,
				meshNamespace:      testNamespace,
				caBundleSecretName: constants.DefaultCABundleSecretName,
				clientSet:          fake.NewSimpleClientset(existingKubeClientsetObjects...),
				// CustomResourceDefinitions belong to the extensionsClientset
				extensionsClientset:        extensionsClientSetFake.NewSimpleClientset(test.existingCustomResourceDefinitions...),
				deleteClusterWideResources: true,
			}

			err = uninstall.run()
			assert.Nil(err)

			// ensure that only the non-OSM CRDs remain in the cluster
			crdsList, err := uninstall.extensionsClientset.ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(crdsList.Items))
			assert.Equal(someOtherCustomResourceDefinition, crdsList.Items[0].Name)

			// ensure that only the non-OSM MutatingWebhookConfigurations remain in the cluster
			mutatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(mutatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, mutatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-OSM ValidatingWebhookConfigurations remain in the cluster
			validatingWebhookConfigurations, err := uninstall.clientSet.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(validatingWebhookConfigurations.Items))
			assert.Equal(someOtherWebhookName, validatingWebhookConfigurations.Items[0].Name)

			// ensure that only the non-OSM Secrets remain in the cluster
			secrets, err := uninstall.clientSet.CoreV1().Secrets(uninstall.meshNamespace).List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(1, len(secrets.Items))
			assert.Equal(someOtherSecretName, secrets.Items[0].Name)

			// ensure that existing namespaces are not deleted as deleting namespaces could be disastrous (for example, if OSM was installed in namespace kube-system)
			namespaces, err := uninstall.clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
			assert.Nil(err)
			assert.Equal(len(test.existingNamespaces), len(namespaces.Items))
		})
	}
}

package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	testRegistry               = "test-registry"
	testRegistrySecret         = "test-registry-secret"
	testOsmImageTag            = "test-tag"
	testVaultHost              = "vault.osm.svc.cluster.local"
	testVaultProtocol          = "http"
	testVaultToken             = "token"
	testVaultRole              = "role"
	testCertManagerIssuerName  = "my-osm-ca"
	testCertManagerIssuerKind  = "ClusterIssuer"
	testCertManagerIssuerGroup = "example.co.uk"
	testCABundleSecretName     = "osm-ca-bundle"
	testRetentionTime          = "5d"
	testEnvoyLogLevel          = "error"
)

var _ = Describe("Running the install command", func() {

	Describe("with default parameters", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()

			installCmd := &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				osmImageTag:                 testOsmImageTag,
				osmImagePullPolicy:          defaultOsmImagePullPolicy,
				certificateManager:          "tresor",
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName,
				enableEgress:                true,
				enablePrometheus:            true,
				enableGrafana:               false,
				enableFluentbit:             false,
				clientSet:                   fakeClientSet,
				envoyLogLevel:               testEnvoyLogLevel,
			}

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("OSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certificateManager": "tresor",
						"certmanager": map[string]interface{}{
							"issuerKind":  "",
							"issuerGroup": "",
							"issuerName":  "",
						},
						"meshName": defaultMeshName,
						"image": map[string]interface{}{
							"registry":   testRegistry,
							"tag":        testOsmImageTag,
							"pullPolicy": defaultOsmImagePullPolicy,
						},
						"serviceCertValidityDuration": "24h",
						"vault": map[string]interface{}{
							"host":     "",
							"protocol": "",
							"token":    "",
							"role":     "",
						},
						"prometheus": map[string]interface{}{
							"retention": map[string]interface{}{
								"time": "5d",
							}},
						"enableDebugServer":              false,
						"enablePermissiveTrafficPolicy":  false,
						"enableBackpressureExperimental": false,
						"enableEgress":                   true,
						"enablePrometheus":               true,
						"enableGrafana":                  false,
						"enableFluentbit":                false,
						"deployJaeger":                   false,
						"envoyLogLevel":                  testEnvoyLogLevel,
						"enforceSingleMesh":              false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("with the default chart from source", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()

			installCmd := &installCmd{
				out:                         out,
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				osmImageTag:                 testOsmImageTag,
				osmImagePullPolicy:          defaultOsmImagePullPolicy,
				certificateManager:          "tresor",
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName,
				enableEgress:                true,
				enablePrometheus:            true,
				enableGrafana:               false,
				enableFluentbit:             false,
				clientSet:                   fakeClientSet,
				envoyLogLevel:               testEnvoyLogLevel,
			}

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("OSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certificateManager": "tresor",
						"certmanager": map[string]interface{}{
							"issuerKind":  "",
							"issuerGroup": "",
							"issuerName":  "",
						},
						"meshName": defaultMeshName,
						"image": map[string]interface{}{
							"registry":   testRegistry,
							"tag":        testOsmImageTag,
							"pullPolicy": defaultOsmImagePullPolicy,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityDuration": "24h",
						"vault": map[string]interface{}{
							"host":     "",
							"protocol": "",
							"token":    "",
							"role":     "",
						},
						"prometheus": map[string]interface{}{
							"retention": map[string]interface{}{
								"time": "5d",
							}},
						"enableDebugServer":              false,
						"enablePermissiveTrafficPolicy":  false,
						"enableBackpressureExperimental": false,
						"enableEgress":                   true,
						"enablePrometheus":               true,
						"enableGrafana":                  false,
						"enableFluentbit":                false,
						"deployJaeger":                   false,
						"envoyLogLevel":                  testEnvoyLogLevel,
						"enforceSingleMesh":              false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("with the vault cert manager", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()

			installCmd := &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				certificateManager:          "vault",
				vaultHost:                   testVaultHost,
				vaultToken:                  testVaultToken,
				vaultRole:                   testVaultRole,
				vaultProtocol:               "http",
				certmanagerIssuerName:       testCertManagerIssuerName,
				certmanagerIssuerKind:       testCertManagerIssuerKind,
				certmanagerIssuerGroup:      testCertManagerIssuerGroup,
				osmImageTag:                 testOsmImageTag,
				osmImagePullPolicy:          defaultOsmImagePullPolicy,
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName,
				enableEgress:                true,
				enablePrometheus:            true,
				enableGrafana:               false,
				enableFluentbit:             false,
				clientSet:                   fakeClientSet,
				envoyLogLevel:               testEnvoyLogLevel,
			}

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("OSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certificateManager": "vault",
						"certmanager": map[string]interface{}{
							"issuerKind":  "ClusterIssuer",
							"issuerGroup": "example.co.uk",
							"issuerName":  "my-osm-ca",
						},
						"meshName": defaultMeshName,
						"image": map[string]interface{}{
							"registry":   testRegistry,
							"tag":        testOsmImageTag,
							"pullPolicy": defaultOsmImagePullPolicy,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityDuration": "24h",
						"vault": map[string]interface{}{
							"host":     testVaultHost,
							"protocol": "http",
							"token":    testVaultToken,
							"role":     testVaultRole,
						},
						"prometheus": map[string]interface{}{
							"retention": map[string]interface{}{
								"time": "5d",
							},
						},
						"enableDebugServer":              false,
						"enablePermissiveTrafficPolicy":  false,
						"enableBackpressureExperimental": false,
						"enableEgress":                   true,
						"enablePrometheus":               true,
						"enableGrafana":                  false,
						"enableFluentbit":                false,
						"deployJaeger":                   false,
						"envoyLogLevel":                  testEnvoyLogLevel,
						"enforceSingleMesh":              false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("without required vault parameters", func() {
		var (
			out    *bytes.Buffer
			store  *storage.Storage
			config *helm.Configuration
			err    error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := &installCmd{
				out:                     out,
				chartPath:               "testdata/test-chart",
				containerRegistry:       testRegistry,
				containerRegistrySecret: testRegistrySecret,
				certificateManager:      "vault",
				meshName:                defaultMeshName,
				enableEgress:            true,
			}

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err).To(MatchError("Missing arguments for certificate-manager vault: [vault-host vault-token]"))
		})
	})

	Describe("with the cert-manager certificate manager", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()

			installCmd := &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				certificateManager:          "cert-manager",
				vaultHost:                   testVaultHost,
				vaultToken:                  testVaultToken,
				vaultRole:                   testVaultRole,
				vaultProtocol:               "http",
				certmanagerIssuerName:       testCertManagerIssuerName,
				certmanagerIssuerKind:       testCertManagerIssuerKind,
				certmanagerIssuerGroup:      testCertManagerIssuerGroup,
				osmImageTag:                 testOsmImageTag,
				osmImagePullPolicy:          defaultOsmImagePullPolicy,
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName,
				enableEgress:                true,
				enablePrometheus:            true,
				enableGrafana:               false,
				enableFluentbit:             false,
				clientSet:                   fakeClientSet,
				envoyLogLevel:               testEnvoyLogLevel,
			}

			err = installCmd.run(config)
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("OSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
		})

		Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certificateManager": "cert-manager",
						"certmanager": map[string]interface{}{
							"issuerKind":  "ClusterIssuer",
							"issuerGroup": "example.co.uk",
							"issuerName":  "my-osm-ca",
						},
						"meshName": defaultMeshName,
						"image": map[string]interface{}{
							"registry":   testRegistry,
							"tag":        testOsmImageTag,
							"pullPolicy": defaultOsmImagePullPolicy,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityDuration": "24h",
						"vault": map[string]interface{}{
							"host":     testVaultHost,
							"protocol": "http",
							"token":    testVaultToken,
							"role":     testVaultRole,
						},
						"prometheus": map[string]interface{}{
							"retention": map[string]interface{}{
								"time": "5d",
							},
						},
						"enableDebugServer":              false,
						"enablePermissiveTrafficPolicy":  false,
						"enableBackpressureExperimental": false,
						"enableEgress":                   true,
						"enablePrometheus":               true,
						"enableGrafana":                  false,
						"enableFluentbit":                false,
						"deployJaeger":                   false,
						"envoyLogLevel":                  testEnvoyLogLevel,
						"enforceSingleMesh":              false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("when a mesh with the given name already exists", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			install       *installCmd
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()
			deploymentSpec := createDeploymentSpec(settings.Namespace(), defaultMeshName)
			fakeClientSet.AppsV1().Deployments(settings.Namespace()).Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})

			install = &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				osmImageTag:                 testOsmImageTag,
				certificateManager:          "tresor",
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName,
				enableEgress:                true,
				clientSet:                   fakeClientSet,
			}

			err = config.Releases.Create(&release.Release{
				Namespace: settings.Namespace(), // should be found in any namespace
				Config: map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"meshName": install.meshName,
					},
				},
				Info: &release.Info{
					// helm list only shows deployed and failed releases by default
					Status: release.StatusDeployed,
				},
			})
			if err != nil {
				panic(err)
			}

			err = install.run(config)
		})

		It("should error", func() {
			Expect(err.Error()).To(Equal(errMeshAlreadyExists(install.meshName).Error()))
		})
	})

	Describe("when a mesh already exists in the given namespace", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			install       *installCmd
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()
			deploymentSpec := createDeploymentSpec(settings.Namespace(), defaultMeshName)
			fakeClientSet.AppsV1().Deployments(settings.Namespace()).Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})

			install = &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				osmImageTag:                 testOsmImageTag,
				certificateManager:          "tresor",
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    defaultMeshName + "-2",
				enableEgress:                true,
				clientSet:                   fakeClientSet,
			}

			err = config.Releases.Create(&release.Release{
				Namespace: settings.Namespace(), // should be found in any namespace
				Config: map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"meshName": install.meshName,
					},
				},
				Info: &release.Info{
					// helm list only shows deployed and failed releases by default
					Status: release.StatusDeployed,
				},
			})
			if err != nil {
				panic(err)
			}

			err = install.run(config)
		})

		It("should error", func() {
			Expect(err.Error()).To(Equal(errNamespaceAlreadyHasController(settings.Namespace()).Error()))
		})
	})

	Describe("when a mesh name is invalid", func() {
		var (
			out     *bytes.Buffer
			store   *storage.Storage
			config  *helm.Configuration
			install *installCmd
			err     error
		)

		BeforeEach(func() {
			out = new(bytes.Buffer)
			store = storage.Init(driver.NewMemory())
			if mem, ok := store.Driver.(*driver.Memory); ok {
				mem.SetNamespace(settings.Namespace())
			}

			config = &helm.Configuration{
				Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			install = &installCmd{
				out:                         out,
				chartPath:                   "testdata/test-chart",
				containerRegistry:           testRegistry,
				containerRegistrySecret:     testRegistrySecret,
				osmImageTag:                 testOsmImageTag,
				certificateManager:          "tresor",
				serviceCertValidityDuration: "24h",
				prometheusRetentionTime:     testRetentionTime,
				meshName:                    "osm!!123456789012345678901234567890123456789012345678901234567890", // >65 characters, contains !
				enableEgress:                true,
			}

			err = install.run(config)
		})

		It("should error", func() {
			Expect(err).To(MatchError("Invalid mesh-name.\nValid mesh-name:\n- must be no longer than 63 characters\n- must consist of alphanumeric characters, '-', '_' or '.'\n- must start and end with an alphanumeric character\nregex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'"))
		})
	})

})

var _ = Describe("Resolving values for install command with vault parameters", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               false,
			enableFluentbit:             false,

			envoyLogLevel: testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  false,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})
})

var _ = Describe("Ensure that grafana is disabled when flag is set to false", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               false,
			enableFluentbit:             false,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  false,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})

})

var _ = Describe("Ensure that grafana is enabled when flag is set to true", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               true,
			enableFluentbit:             false,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  true,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})

})

var _ = Describe("Ensure that prometheus is disabled when flag is set to false", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            false,
			enableGrafana:               false,
			enableFluentbit:             false,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               false,
				"enableGrafana":                  false,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})
})

var _ = Describe("Ensure that fluentbit is disabled when flag is set to false", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               false,
			enableFluentbit:             false,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  false,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})

})

var _ = Describe("Ensure that fluentbit is enabled when flag is set to true", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "vault",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               false,
			enableFluentbit:             true,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "vault",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  false,
				"enableFluentbit":                true,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})
})

var _ = Describe("Resolving values for install command with cert-manager parameters", func() {
	var (
		vals map[string]interface{}
		err  error
	)

	BeforeEach(func() {
		installCmd := &installCmd{
			containerRegistry:           testRegistry,
			containerRegistrySecret:     testRegistrySecret,
			certificateManager:          "cert-manager",
			vaultHost:                   testVaultHost,
			vaultProtocol:               testVaultProtocol,
			certmanagerIssuerName:       testCertManagerIssuerName,
			certmanagerIssuerKind:       testCertManagerIssuerKind,
			certmanagerIssuerGroup:      testCertManagerIssuerGroup,
			vaultToken:                  testVaultToken,
			vaultRole:                   testVaultRole,
			osmImageTag:                 testOsmImageTag,
			osmImagePullPolicy:          defaultOsmImagePullPolicy,
			serviceCertValidityDuration: "24h",
			prometheusRetentionTime:     testRetentionTime,
			meshName:                    defaultMeshName,
			enableEgress:                true,
			enablePrometheus:            true,
			enableGrafana:               false,
			enableFluentbit:             false,
			envoyLogLevel:               testEnvoyLogLevel,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certificateManager": "cert-manager",
				"certmanager": map[string]interface{}{
					"issuerKind":  "ClusterIssuer",
					"issuerGroup": "example.co.uk",
					"issuerName":  "my-osm-ca",
				},
				"meshName": defaultMeshName,
				"image": map[string]interface{}{
					"registry":   testRegistry,
					"tag":        testOsmImageTag,
					"pullPolicy": defaultOsmImagePullPolicy,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityDuration": "24h",
				"vault": map[string]interface{}{
					"host":     testVaultHost,
					"protocol": "http",
					"token":    testVaultToken,
					"role":     testVaultRole,
				},
				"prometheus": map[string]interface{}{
					"retention": map[string]interface{}{
						"time": "5d",
					},
				},
				"enableDebugServer":              false,
				"enablePermissiveTrafficPolicy":  false,
				"enableBackpressureExperimental": false,
				"enableEgress":                   true,
				"enablePrometheus":               true,
				"enableGrafana":                  false,
				"enableFluentbit":                false,
				"deployJaeger":                   false,
				"envoyLogLevel":                  testEnvoyLogLevel,
				"enforceSingleMesh":              false,
			}}))
	})
})

var _ = Describe("Resolving values for egress option", func() {
	Context("Test enableEgress chart value with install cli option", func() {
		It("Should disable egress in the Helm chart", func() {
			installCmd := &installCmd{
				enableEgress: false,
			}

			vals, err := installCmd.resolveValues()
			Expect(err).NotTo(HaveOccurred())

			enableEgressVal := vals["OpenServiceMesh"].(map[string]interface{})["enableEgress"]
			Expect(enableEgressVal).To(BeFalse())
		})

		It("Should enable egress in the Helm chart", func() {
			installCmd := &installCmd{
				enableEgress: true,
			}

			vals, err := installCmd.resolveValues()
			Expect(err).NotTo(HaveOccurred())

			enableEgressVal := vals["OpenServiceMesh"].(map[string]interface{})["enableEgress"]
			Expect(enableEgressVal).To(BeTrue())
		})
	})
})

var _ = Describe("Test envoy log level types", func() {
	Context("Test envoyLogLevel chart value with install cli option", func() {
		It("Should correctly resolve envoyLogLevel when specified", func() {
			installCmd := &installCmd{
				envoyLogLevel: testEnvoyLogLevel,
			}

			vals, err := installCmd.resolveValues()
			Expect(err).NotTo(HaveOccurred())

			logLevel := vals["OpenServiceMesh"].(map[string]interface{})["envoyLogLevel"]
			Expect(logLevel).To(Equal(testEnvoyLogLevel))
		})
	})

	Context("Test isValidEnvoyLogLevel", func() {
		It("Should validate if the specified envoy log level is supported", func() {
			err := isValidEnvoyLogLevel("error")
			Expect(err).NotTo(HaveOccurred())

			err = isValidEnvoyLogLevel("off")
			Expect(err).NotTo(HaveOccurred())

			err = isValidEnvoyLogLevel("warn")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should correctly error for invalid envoy log level", func() {
			err := isValidEnvoyLogLevel("tracing")
			Expect(err).To(HaveOccurred())

			err = isValidEnvoyLogLevel("warns")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Test osm image pull policy cli option", func() {
	It("Should correctly resolve the pull policy option to chart values", func() {
		installCmd := &installCmd{
			osmImagePullPolicy: "IfNotPresent",
		}

		vals, err := installCmd.resolveValues()
		Expect(err).NotTo(HaveOccurred())

		pullPolicy := vals["OpenServiceMesh"].(map[string]interface{})["image"].(map[string]interface{})["pullPolicy"]
		Expect(pullPolicy).To(Equal("IfNotPresent"))
	})

	It("Should correctly resolve the pull policy option to chart values", func() {
		installCmd := &installCmd{
			osmImagePullPolicy: "Always",
		}

		vals, err := installCmd.resolveValues()
		Expect(err).NotTo(HaveOccurred())

		pullPolicy := vals["OpenServiceMesh"].(map[string]interface{})["image"].(map[string]interface{})["pullPolicy"]
		Expect(pullPolicy).To(Equal("Always"))
	})
})

var _ = Describe("Resolving values for service cert validity duration", func() {
	It("Should correctly resolve the validity duration", func() {
		installCmd := &installCmd{
			serviceCertValidityDuration: "24h",
		}

		vals, err := installCmd.resolveValues()
		Expect(err).NotTo(HaveOccurred())

		actual := vals["OpenServiceMesh"].(map[string]interface{})["serviceCertValidityDuration"]
		Expect(actual).To(Equal("24h"))
	})

	It("Should correctly resolve the validity duration", func() {
		installCmd := &installCmd{
			serviceCertValidityDuration: "1h30m",
		}

		vals, err := installCmd.resolveValues()
		Expect(err).NotTo(HaveOccurred())

		actual := vals["OpenServiceMesh"].(map[string]interface{})["serviceCertValidityDuration"]
		Expect(actual).To(Equal("1h30m"))
	})
})

func TestEnforceSingleMesh(t *testing.T) {
	assert := assert.New(t)

	out := new(bytes.Buffer)
	store := storage.Init(driver.NewMemory())
	if mem, ok := store.Driver.(*driver.Memory); ok {
		mem.SetNamespace(settings.Namespace())
	}

	config := &helm.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(format string, v ...interface{}) {},
	}

	fakeClientSet := fake.NewSimpleClientset()

	install := &installCmd{
		out:                         out,
		chartPath:                   "testdata/test-chart",
		containerRegistry:           testRegistry,
		osmImageTag:                 testOsmImageTag,
		osmImagePullPolicy:          defaultOsmImagePullPolicy,
		certificateManager:          "tresor",
		serviceCertValidityDuration: "24h",
		prometheusRetentionTime:     testRetentionTime,
		meshName:                    defaultMeshName,
		enableEgress:                true,
		enablePrometheus:            true,
		enableGrafana:               false,
		clientSet:                   fakeClientSet,
		envoyLogLevel:               testEnvoyLogLevel,
		enforceSingleMesh:           true,
	}

	err := install.run(config)
	assert.Nil(err)
	assert.Equal(out.String(), "OSM installed successfully in namespace [osm-system] with mesh name [osm]\n")
}

func TestEnforceSingleMeshRejectsNewMesh(t *testing.T) {
	assert := assert.New(t)

	out := new(bytes.Buffer)
	store := storage.Init(driver.NewMemory())
	if mem, ok := store.Driver.(*driver.Memory); ok {
		mem.SetNamespace(settings.Namespace())
	}

	config := &helm.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(format string, v ...interface{}) {},
	}

	fakeClientSet := fake.NewSimpleClientset()

	labelMap := make(map[string]string)
	labelMap["meshName"] = defaultMeshName
	labelMap["app"] = constants.OSMControllerName
	labelMap["enforceSingleMesh"] = "true"

	deploymentSpec := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.OSMControllerName,
			Namespace: settings.Namespace() + "-existing",
			Labels:    labelMap,
		},
	}
	_, err := fakeClientSet.AppsV1().Deployments(settings.Namespace()+"-existing").Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})
	assert.Nil(err)

	install := &installCmd{
		out:                         out,
		chartPath:                   "testdata/test-chart",
		containerRegistry:           testRegistry,
		osmImageTag:                 testOsmImageTag,
		osmImagePullPolicy:          defaultOsmImagePullPolicy,
		certificateManager:          "tresor",
		serviceCertValidityDuration: "24h",
		prometheusRetentionTime:     testRetentionTime,
		meshName:                    defaultMeshName + "-2",
		enableEgress:                true,
		enablePrometheus:            true,
		enableGrafana:               false,
		clientSet:                   fakeClientSet,
		envoyLogLevel:               testEnvoyLogLevel,
	}

	err = install.run(config)
	assert.NotNil(err)
	assert.True(strings.Contains(err.Error(), "Cannot install mesh [osm-2]. Existing mesh [osm] enforces single mesh cluster"))
}

func TestEnforceSingleMeshWithExistingMesh(t *testing.T) {
	assert := assert.New(t)

	out := new(bytes.Buffer)
	store := storage.Init(driver.NewMemory())
	if mem, ok := store.Driver.(*driver.Memory); ok {
		mem.SetNamespace(settings.Namespace())
	}

	config := &helm.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(format string, v ...interface{}) {},
	}

	fakeClientSet := fake.NewSimpleClientset()

	deploymentSpec := createDeploymentSpec(settings.Namespace()+"-existing", defaultMeshName)
	_, err := fakeClientSet.AppsV1().Deployments(settings.Namespace()+"-existing").Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})
	assert.Nil(err)

	install := &installCmd{
		out:                         out,
		chartPath:                   "testdata/test-chart",
		containerRegistry:           testRegistry,
		osmImageTag:                 testOsmImageTag,
		osmImagePullPolicy:          defaultOsmImagePullPolicy,
		certificateManager:          "tresor",
		serviceCertValidityDuration: "24h",
		prometheusRetentionTime:     testRetentionTime,
		meshName:                    defaultMeshName + "-2",
		enableEgress:                true,
		enablePrometheus:            true,
		enableGrafana:               false,
		clientSet:                   fakeClientSet,
		envoyLogLevel:               testEnvoyLogLevel,
		enforceSingleMesh:           true,
	}

	err = install.run(config)
	assert.NotNil(err)
	assert.True(strings.Contains(err.Error(), "Meshes already exist in cluster. Cannot enforce single mesh cluster"))
}

func createDeploymentSpec(namespace, meshName string) *v1.Deployment {
	labelMap := make(map[string]string)
	if meshName != "" {
		labelMap["meshName"] = meshName
		labelMap["app"] = constants.OSMControllerName
	}
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.OSMControllerName,
			Namespace: namespace,
			Labels:    labelMap,
		},
	}
}

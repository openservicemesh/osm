package main

import (
	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var (
	testRegistry       = "test-registry"
	testRegistrySecret = "test-registry-secret"
	testOsmImageTag    = "test-tag"
	testVaultHost      = "vault.osm.svc.cluster.local"
	testVaultProtocol  = "http"
	testVaultToken     = "token"
	testVaultRole      = "role"
	testRetentionTime  = "5d"
)

var _ = Describe("Running the install command", func() {

	Describe("with default parameters", func() {
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
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := &installCmd{
				out:                        out,
				chartPath:                  "testdata/test-chart",
				containerRegistry:          testRegistry,
				containerRegistrySecret:    testRegistrySecret,
				osmImageTag:                testOsmImageTag,
				certManager:                "tresor",
				serviceCertValidityMinutes: 1,
				prometheusRetentionTime:    testRetentionTime,
				meshName:                   "osm",
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
				rel, err = config.Releases.Get(settings.Namespace(), 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certManager": "tresor",
						"meshName":    "osm",
						"image": map[string]interface{}{
							"registry": testRegistry,
							"tag":      testOsmImageTag,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityMinutes": int64(1),
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
						"enableDebugServer":             false,
						"disableSMIAccessControlPolicy": false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("with the default chart from source", func() {
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
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			installCmd := &installCmd{
				out:                        out,
				containerRegistry:          testRegistry,
				containerRegistrySecret:    testRegistrySecret,
				osmImageTag:                testOsmImageTag,
				certManager:                "tresor",
				serviceCertValidityMinutes: 1,
				prometheusRetentionTime:    testRetentionTime,
				meshName:                   "osm",
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
				rel, err = config.Releases.Get(settings.Namespace(), 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certManager": "tresor",
						"meshName":    "osm",
						"image": map[string]interface{}{
							"registry": testRegistry,
							"tag":      testOsmImageTag,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityMinutes": int64(1),
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
						"enableDebugServer":             false,
						"disableSMIAccessControlPolicy": false,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("with the vault cert manager", func() {
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
				out:                        out,
				chartPath:                  "testdata/test-chart",
				containerRegistry:          testRegistry,
				containerRegistrySecret:    testRegistrySecret,
				certManager:                "vault",
				vaultHost:                  testVaultHost,
				vaultToken:                 testVaultToken,
				vaultRole:                  testVaultRole,
				vaultProtocol:              "http",
				osmImageTag:                testOsmImageTag,
				serviceCertValidityMinutes: 1,
				prometheusRetentionTime:    testRetentionTime,
				meshName:                   "osm",
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
				rel, err = config.Releases.Get(settings.Namespace(), 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"certManager": "vault",
						"meshName":    "osm",
						"image": map[string]interface{}{
							"registry": testRegistry,
							"tag":      testOsmImageTag,
						},
						"imagePullSecrets": []interface{}{
							map[string]interface{}{
								"name": testRegistrySecret,
							},
						},
						"serviceCertValidityMinutes": int64(1),
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
						"enableDebugServer":             false,
						"disableSMIAccessControlPolicy": false,
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
				certManager:             "vault",
			}

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err).To(MatchError("Missing arguments for cert-manager vault: [vault-host vault-token]"))
		})
	})

	Describe("when a mesh with the given name already exists", func() {
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
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			install = &installCmd{
				out:                        out,
				chartPath:                  "testdata/test-chart",
				containerRegistry:          testRegistry,
				containerRegistrySecret:    testRegistrySecret,
				osmImageTag:                testOsmImageTag,
				certManager:                "tresor",
				serviceCertValidityMinutes: 1,
				prometheusRetentionTime:    testRetentionTime,
				meshName:                   "osm",
			}

			err = config.Releases.Create(&release.Release{
				Namespace: "not-" + settings.Namespace(), // should be found in any namespace
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
			Expect(err).To(MatchError(errMeshAlreadyExists(install.meshName)))
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
			containerRegistry:          testRegistry,
			containerRegistrySecret:    testRegistrySecret,
			certManager:                "vault",
			vaultHost:                  testVaultHost,
			vaultProtocol:              testVaultProtocol,
			vaultToken:                 testVaultToken,
			vaultRole:                  testVaultRole,
			osmImageTag:                testOsmImageTag,
			serviceCertValidityMinutes: 1,
			prometheusRetentionTime:    testRetentionTime,
		}

		vals, err = installCmd.resolveValues()
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("should resolve correctly", func() {
		Expect(vals).To(BeEquivalentTo(map[string]interface{}{
			"OpenServiceMesh": map[string]interface{}{
				"certManager": "vault",
				"meshName":    "",
				"image": map[string]interface{}{
					"registry": testRegistry,
					"tag":      testOsmImageTag,
				},
				"imagePullSecrets": []interface{}{
					map[string]interface{}{
						"name": testRegistrySecret,
					},
				},
				"serviceCertValidityMinutes": int64(1),
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
				"enableDebugServer":             false,
				"disableSMIAccessControlPolicy": false,
			}}))
	})
})

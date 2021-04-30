package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	tassert "github.com/stretchr/testify/assert"

	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	yaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"

	"github.com/openservicemesh/osm/pkg/constants"
)

var (
	testRegistry           = "test-registry"
	testRegistrySecret     = "test-registry-secret"
	testOsmImageTag        = "test-tag"
	testVaultHost          = "vault.osm.svc.cluster.local"
	testVaultToken         = "token"
	testRetentionTime      = "5d"
	testControllerLogLevel = "info"
	testChartPath          = "testdata/test-chart"
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

			installCmd := getDefaultInstallCmd(out)

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
				defaultValues := getDefaultValues()
				Expect(rel.Config).To(BeEquivalentTo(defaultValues))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		})
	})

	Describe("with a default Helm chart", func() {
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

			installCmd := getDefaultInstallCmd(out)
			installCmd.chartPath = "testdata/test-chart"

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
				defaultValues := getDefaultValues()
				Expect(rel.Config).To(BeEquivalentTo(defaultValues))
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

			installCmd := getDefaultInstallCmd(out)
			installCmd.certificateManager = "vault"
			installCmd.vaultHost = testVaultHost
			installCmd.vaultToken = testVaultToken

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
				expectedValues := getDefaultValues()
				valuesConfig := []string{
					fmt.Sprintf("OpenServiceMesh.certificateManager=%s", "vault"),
					fmt.Sprintf("OpenServiceMesh.vault.host=%s", testVaultHost),
					fmt.Sprintf("OpenServiceMesh.vault.token=%s", testVaultToken),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(rel.Config).To(BeEquivalentTo(expectedValues))
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

			installCmd := getDefaultInstallCmd(out)
			installCmd.certificateManager = "vault"

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err).To(MatchError("Missing arguments for certificate-manager vault: [vault-host vault-token]"))
		})
	})

	Describe("with the cert-manager certificate manager", func() {
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

			installCmd := getDefaultInstallCmd(out)
			installCmd.certificateManager = "cert-manager"

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
				expectedValues := getDefaultValues()
				valuesConfig := []string{
					fmt.Sprintf("OpenServiceMesh.certificateManager=%s", "cert-manager"),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(rel.Config).To(BeEquivalentTo(expectedValues))
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
			installCmd    installCmd
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
			_, err = fakeClientSet.AppsV1().Deployments(settings.Namespace()).Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			installCmd = getDefaultInstallCmd(out)
			// Use the client set with the existing mesh deployment
			installCmd.clientSet = fakeClientSet

			err = config.Releases.Create(&release.Release{
				Namespace: settings.Namespace(), // should be found in any namespace
				Config: map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"meshName": installCmd.meshName,
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

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMeshAlreadyExists(installCmd.meshName).Error()))
		})
	})

	Describe("when a mesh already exists in the given namespace", func() {
		var (
			out           *bytes.Buffer
			store         *storage.Storage
			config        *helm.Configuration
			installCmd    installCmd
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
			_, err = fakeClientSet.AppsV1().Deployments(settings.Namespace()).Create(context.TODO(), deploymentSpec, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			installCmd = getDefaultInstallCmd(out)
			installCmd.meshName = defaultMeshName + "-2" //use different name than pre-existing mesh
			installCmd.clientSet = fakeClientSet

			// Create pre-existing mesh
			err = config.Releases.Create(&release.Release{
				Namespace: settings.Namespace(), // should be found in any namespace
				Config: map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"meshName": defaultMeshName,
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

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err.Error()).To(Equal(errNamespaceAlreadyHasController(settings.Namespace()).Error()))
		})
	})

	Describe("when a mesh name is invalid", func() {
		var (
			out        *bytes.Buffer
			store      *storage.Storage
			config     *helm.Configuration
			installCmd installCmd
			err        error
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

			installCmd = getDefaultInstallCmd(out)
			installCmd.meshName = "osm!!123456789012345678901234567890123456789012345678901234567890" // >65 characters, contains !

			err = installCmd.run(config)
		})

		It("should error", func() {
			Expect(err).To(MatchError("Invalid mesh-name.\nValid mesh-name:\n- must be no longer than 63 characters\n- must consist of alphanumeric characters, '-', '_' or '.'\n- must start and end with an alphanumeric character\nregex used for validation is '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'"))
		})
	})

})

var _ = Describe("deployPrometheus is true", func() {
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

		installCmd := getDefaultInstallCmd(out)
		installCmd.deployPrometheus = true
		installCmd.enablePrometheusScraping = false

		err = installCmd.run(config)
	})

	It("should not error", func() {
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(Equal("Prometheus scraping is disabled. To enable it, set prometheus_scraping in osm-system/osm-config to true.\nOSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
	})
})

func TestResolveValues(t *testing.T) {
	tests := []struct {
		name        string
		installCmd  installCmd
		expected    map[string]interface{}
		expectedErr error
	}{
		{
			name: "default",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)

				// Fill out fields which are empty by default
				installCmd.containerRegistrySecret = testRegistrySecret
				installCmd.vaultHost = testVaultHost
				installCmd.vaultToken = testVaultToken
				return installCmd
			}(),
			expected: func() map[string]interface{} {
				expectedValues := getDefaultValues()

				// Fill out fields which are empty by default
				valuesConfig := []string{
					fmt.Sprintf("OpenServiceMesh.imagePullSecrets[0].name=%s", testRegistrySecret),
					fmt.Sprintf("OpenServiceMesh.vault.host=%s", testVaultHost),
					fmt.Sprintf("OpenServiceMesh.vault.token=%s", testVaultToken),
				}
				for _, val := range valuesConfig {
					// parses Helm strvals line and merges into a map
					err := strvals.ParseInto(val, expectedValues)
					tassert.Nil(t, err)
				}
				return expectedValues
			}(),
		},
		{
			name: "--set creates additional values",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"new=from set", "key1=val1,key2=val2"}
				return installCmd
			}(),
			expected: func() map[string]interface{} {
				vals := getDefaultValues()
				vals["new"] = "from set"
				vals["key1"] = "val1"
				vals["key2"] = "val2"
				return vals
			}(),
		},
		{
			name: "--set for an existing parameter as no effect",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"OpenServiceMesh.meshName=set"}
				return installCmd
			}(),
			expected: getDefaultValues(),
		},
		{
			name: "invalid --set format",
			installCmd: func() installCmd {
				installCmd := getDefaultInstallCmd(ioutil.Discard)
				installCmd.setOptions = []string{"can't set this"}
				return installCmd
			}(),
			expectedErr: errors.New("invalid format for --set: key \"can't set this\" has no value"),
		},
	}

	for idx, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := tassert.New(t)
			actual, err := test.installCmd.resolveValues()
			if err != nil && test.expectedErr != nil {
				assert.Equal(err.Error(), test.expectedErr.Error())
			} else {
				assert.Equal(err, test.expectedErr)
			}
			assert.Equal(actual, test.expected, "Test at index %d failed", idx)
		})
	}
}

func TestEnforceSingleMesh(t *testing.T) {
	assert := tassert.New(t)

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
		out:                     out,
		chartPath:               testChartPath,
		containerRegistry:       testRegistry,
		osmImageTag:             testOsmImageTag,
		osmImagePullPolicy:      defaultOsmImagePullPolicy,
		certificateManager:      "tresor",
		prometheusRetentionTime: testRetentionTime,
		meshName:                defaultMeshName,
		deployGrafana:           false,
		clientSet:               fakeClientSet,
		controllerLogLevel:      testControllerLogLevel,
		enforceSingleMesh:       true,
	}

	err := install.run(config)
	assert.Nil(err)
	assert.Equal(out.String(), "OSM installed successfully in namespace [osm-system] with mesh name [osm]\n")
}

func TestEnforceSingleMeshRejectsNewMesh(t *testing.T) {
	assert := tassert.New(t)

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
		out:                     out,
		chartPath:               testChartPath,
		containerRegistry:       testRegistry,
		osmImageTag:             testOsmImageTag,
		osmImagePullPolicy:      defaultOsmImagePullPolicy,
		certificateManager:      "tresor",
		prometheusRetentionTime: testRetentionTime,
		meshName:                defaultMeshName + "-2",
		deployPrometheus:        true,
		deployGrafana:           false,
		clientSet:               fakeClientSet,
		controllerLogLevel:      testControllerLogLevel,
	}

	err = install.run(config)
	assert.NotNil(err)
	assert.True(strings.Contains(err.Error(), "Cannot install mesh [osm-2]. Existing mesh [osm] enforces single mesh cluster"))
}

func TestEnforceSingleMeshWithExistingMesh(t *testing.T) {
	assert := tassert.New(t)

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
		out:                     out,
		chartPath:               testChartPath,
		containerRegistry:       testRegistry,
		osmImageTag:             testOsmImageTag,
		osmImagePullPolicy:      defaultOsmImagePullPolicy,
		certificateManager:      "tresor",
		prometheusRetentionTime: testRetentionTime,
		meshName:                defaultMeshName + "-2",
		deployPrometheus:        true,
		deployGrafana:           false,
		clientSet:               fakeClientSet,
		controllerLogLevel:      testControllerLogLevel,
		enforceSingleMesh:       true,
	}

	err = install.run(config)
	assert.NotNil(err)
	assert.Equal(err, errAlreadyExists)
}

func TestReadMeshConfigFile(t *testing.T) {
	assert := tassert.New(t)

	meshConfigContent := `
apiVersion: config.openservicemesh.io/v1alpha1
kind: MeshConfig
metadata:
  name: test-osm-mesh-config
spec:
  sidecar:
    enable_privileged_init_container: false
    log_level: error
    max_data_plane_connections: 0
  traffic:
    egress: false
    use_https_ingress: false
    enable_permissive_traffic_policy_mode: false
  observability:
    enable_debug_server: true
    prometheus_scraping: True
    tracing:
      enable: false
  certificate:
    service_cert_validity_duration: 24h
`

	var meshConfig v1alpha1.MeshConfig

	err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(meshConfigContent), 1024).Decode(&meshConfig)

	assert.Equal(meshConfig.Name, "test-osm-mesh-config")
	assert.Nil(err)
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

func getDefaultInstallCmd(writer io.Writer) installCmd {
	installCmd := installCmd{
		out:                     writer,
		certificateManager:      defaultCertificateManager,
		certManagerIssuerGroup:  defaultCertManagerIssuerGroup,
		certManagerIssuerKind:   defaultCertManagerIssuerKind,
		certManagerIssuerName:   defaultCertManagerIssuerName,
		chartPath:               defaultChartPath,
		containerRegistry:       defaultContainerRegistry,
		containerRegistrySecret: defaultContainerRegistrySecret,
		meshName:                defaultMeshName,
		osmImagePullPolicy:      defaultOsmImagePullPolicy,
		osmImageTag:             defaultOsmImageTag,
		prometheusRetentionTime: defaultPrometheusRetentionTime,
		vaultHost:               defaultVaultHost,
		vaultProtocol:           defaultVaultProtocol,
		vaultToken:              defaultVaultToken,
		vaultRole:               defaultVaultRole,
		controllerLogLevel:      defaultControllerLogLevel,
		clientSet:               fake.NewSimpleClientset(),
		deployPrometheus:        defaultDeployPrometheus,
		deployGrafana:           defaultDeployGrafana,
		enableFluentbit:         defaultEnableFluentbit,
		deployJaeger:            defaultDeployJaeger,
		enforceSingleMesh:       defaultEnforceSingleMesh,
	}

	return installCmd
}

func getDefaultValues() map[string]interface{} {
	return map[string]interface{}{
		"OpenServiceMesh": map[string]interface{}{
			"certificateManager": defaultCertificateManager,
			"certmanager": map[string]interface{}{
				"issuerKind":  defaultCertManagerIssuerKind,
				"issuerGroup": defaultCertManagerIssuerGroup,
				"issuerName":  defaultCertManagerIssuerName,
			},
			"meshName": defaultMeshName,
			"image": map[string]interface{}{
				"registry":   defaultContainerRegistry,
				"tag":        defaultOsmImageTag,
				"pullPolicy": defaultOsmImagePullPolicy,
			},
			"vault": map[string]interface{}{
				"host":     defaultVaultHost,
				"protocol": defaultVaultProtocol,
				"token":    defaultVaultToken,
				"role":     defaultVaultRole,
			},
			"prometheus": map[string]interface{}{
				"retention": map[string]interface{}{
					"time": defaultPrometheusRetentionTime,
				}},
			"deployPrometheus":   defaultDeployPrometheus,
			"deployGrafana":      defaultDeployGrafana,
			"enableFluentbit":    defaultEnableFluentbit,
			"deployJaeger":       defaultDeployJaeger,
			"controllerLogLevel": testControllerLogLevel,
			"enforceSingleMesh":  defaultEnforceSingleMesh,
		}}
}

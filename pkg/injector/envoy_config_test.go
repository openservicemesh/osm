package injector

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
	"github.com/openservicemesh/osm/pkg/version"
)

var _ = Describe("Test functions creating Envoy bootstrap configuration", func() {
	const (
		containerName = "-container-name-"
		envoyImage    = "-envoy-image-"
		nodeID        = "-node-id-"
		clusterID     = "-cluster-id-"

		// This file contains the Bootstrap YAML generated for all of the Envoy proxies in OSM.
		// This is provisioned by the MutatingWebhook during the addition of a sidecar
		// to every new Pod that is being created in a namespace participating in the service mesh.
		// We deliberately leave this entire string literal here to document and visualize what the
		// generated YAML looks like!
		expectedEnvoyBootstrapConfigFileName        = "expected_envoy_bootstrap_config.yaml"
		actualGeneratedEnvoyBootstrapConfigFileName = "actual_envoy_bootstrap_config.yaml"

		expectedXDSClusterWithoutProbesFileName = "expected_xds_cluster_without_probes.yaml"
		actualXDSClusterWithoutProbesFileName   = "actual_xds_cluster_without_probes.yaml"

		expectedXDSClusterWithProbesFileName = "expected_xds_cluster_with_probes.yaml"
		actualXDSClusterWithProbesFileName   = "actual_xds_cluster_with_probes.yaml"

		expectedXDSStaticResourcesWithProbesFileName = "expected_xds_static_resources_with_probes.yaml"
		actualXDSStaticResourcesWithProbesFileName   = "actual_xds_static_resources_with_probes.yaml"

		// All the YAML files listed above are in this sub-directory
		directoryForYAMLFiles = "test_fixtures"
	)

	isTrue := true
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			OwnerReferences: []metav1.OwnerReference{
				{
					Name: "not-me",
					Kind: "still not me",
				},
				{
					Name:       "workload-name",
					Kind:       "workload-kind",
					Controller: &isTrue,
				},
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: "svcacc",
		},
	}

	cert := tresor.NewFakeCertificate()
	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	originalHealthProbes := healthProbes{
		liveness:  &healthProbe{path: "/liveness", port: 81},
		readiness: &healthProbe{path: "/readiness", port: 82},
		startup:   &healthProbe{path: "/startup", port: 83},
	}

	expectedRewrittenContainerPorts := []corev1.ContainerPort{
		{Name: "proxy-admin", HostPort: 0, ContainerPort: 15000, Protocol: "", HostIP: ""},
		{Name: "proxy-inbound", HostPort: 0, ContainerPort: 15003, Protocol: "", HostIP: ""},
		{Name: "proxy-metrics", HostPort: 0, ContainerPort: 15010, Protocol: "", HostIP: ""},
		{Name: "liveness-port", HostPort: 0, ContainerPort: 15901, Protocol: "", HostIP: ""},
		{Name: "readiness-port", HostPort: 0, ContainerPort: 15902, Protocol: "", HostIP: ""},
		{Name: "startup-port", HostPort: 0, ContainerPort: 15903, Protocol: "", HostIP: ""},
	}

	getExpectedEnvoyYAML := func(filename string) string {
		expectedEnvoyConfig, err := ioutil.ReadFile(filepath.Clean(path.Join(directoryForYAMLFiles, filename)))
		if err != nil {
			log.Err(err).Msgf("Error reading expected Envoy bootstrap YAML from file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
		return string(expectedEnvoyConfig)
	}

	saveActualEnvoyYAML := func(filename string, actualContent []byte) {
		err := ioutil.WriteFile(filepath.Clean(path.Join(directoryForYAMLFiles, filename)), actualContent, 0600)
		if err != nil {
			log.Err(err).Msgf("Error writing actual Envoy Cluster XDS YAML to file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
	}

	marshalAndSaveToFile := func(someStruct interface{}, filename string) string {
		yamlBytes, err := yaml.Marshal(someStruct)
		Expect(err).ToNot(HaveOccurred())
		saveActualEnvoyYAML(filename, yamlBytes)
		return string(yamlBytes)
	}

	probes := healthProbes{
		liveness:  &healthProbe{path: "/liveness", port: 81},
		readiness: &healthProbe{path: "/readiness", port: 82},
		startup:   &healthProbe{path: "/startup", port: 83},
	}

	config := envoyBootstrapConfigMeta{
		RootCert: base64.StdEncoding.EncodeToString(cert.GetIssuingCA()),
		Cert:     base64.StdEncoding.EncodeToString(cert.GetCertificateChain()),
		Key:      base64.StdEncoding.EncodeToString(cert.GetPrivateKey()),

		EnvoyAdminPort: 15000,

		XDSClusterName: "osm-controller",
		XDSHost:        "osm-controller.b.svc.cluster.local",
		XDSPort:        15128,

		OriginalHealthProbes: probes,
	}

	Context("Test getEnvoyConfigYAML()", func() {
		It("creates Envoy bootstrap config", func() {
			config.OriginalHealthProbes = probes
			actual, err := getEnvoyConfigYAML(config, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())
			saveActualEnvoyYAML(actualGeneratedEnvoyBootstrapConfigFileName, actual)

			expectedEnvoyConfig := getExpectedEnvoyYAML(expectedEnvoyBootstrapConfigFileName)

			Expect(string(actual)).To(Equal(expectedEnvoyConfig),
				fmt.Sprintf("Compare files %s and %s\nExpected:\n%s\nActual:\n%s\n",
					expectedEnvoyBootstrapConfigFileName, actualGeneratedEnvoyBootstrapConfigFileName, expectedEnvoyConfig, string(actual)))
		})

		It("Creates bootstrap config for the Envoy proxy", func() {
			wh := &mutatingWebhook{
				kubeClient:          fake.NewSimpleClientset(),
				kubeController:      k8s.NewMockController(gomock.NewController(GinkgoT())),
				nonInjectNamespaces: mapset.NewSet(),
				meshName:            "some-mesh",
			}
			name := uuid.New().String()
			namespace := "a"
			osmNamespace := "b"

			secret, err := wh.createEnvoyBootstrapConfig(name, namespace, osmNamespace, cert, probes)
			Expect(err).ToNot(HaveOccurred())

			expected := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels: map[string]string{
						constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
						constants.OSMAppInstanceLabelKey: "some-mesh",
						constants.OSMAppVersionLabelKey:  version.Version,
					},
				},
				Data: map[string][]byte{
					envoyBootstrapConfigFile: []byte(getExpectedEnvoyYAML(expectedEnvoyBootstrapConfigFileName)),
				},
			}

			// Contains only the "bootstrap.yaml" key
			Expect(len(secret.Data)).To(Equal(1))

			Expect(secret.Data[envoyBootstrapConfigFile]).To(Equal(expected.Data[envoyBootstrapConfigFile]),
				fmt.Sprintf("Expected YAML: %s;\nActual YAML: %s\n", expected.Data, secret.Data))

			// Now check the entire struct
			Expect(*secret).To(Equal(expected))
		})
	})

	Context("Test getXdsCluster()", func() {
		It("creates XDS Cluster struct without health probes", func() {
			config.OriginalHealthProbes = probes
			actual := getXdsCluster(config)

			// The "marshalAndSaveToFile" function converts the complex struct into a human readable text, which helps us spot the
			// difference when there is a discrepancy.
			expectedYAML := getExpectedEnvoyYAML(expectedXDSClusterWithoutProbesFileName)
			actualYAML := marshalAndSaveToFile(actual, actualXDSClusterWithoutProbesFileName)

			Expect(actualYAML).To(Equal(expectedYAML),
				fmt.Sprintf("Compare files %s and %s\nExpected: %s\nActual struct: %s",
					expectedXDSClusterWithoutProbesFileName, actualXDSClusterWithoutProbesFileName, expectedYAML, actualYAML))
		})

		It("creates XDS Cluster struct with health probes", func() {
			config.OriginalHealthProbes = probes
			actual := getXdsCluster(config)

			// The "marshalAndSaveToFile" function converts the complex struct into a human readable text, which helps us spot the
			// difference when there is a discrepancy.
			expectedYAML := getExpectedEnvoyYAML(expectedXDSClusterWithProbesFileName)
			actualYAML := marshalAndSaveToFile(actual, actualXDSClusterWithProbesFileName)

			Expect(actualYAML).To(Equal(expectedYAML),
				fmt.Sprintf("Compare files %s and %s\nExpected: %s\nActual struct: %s",
					expectedXDSClusterWithProbesFileName, actualXDSClusterWithProbesFileName, expectedYAML, actualYAML))
		})
	})

	Context("Test getStaticResources()", func() {
		It("Creates static_resources Envoy struct", func() {
			config.OriginalHealthProbes = healthProbes{}
			actual := getStaticResources(config)

			expectedYAML := getExpectedEnvoyYAML(expectedXDSStaticResourcesWithProbesFileName)
			actualYAML := marshalAndSaveToFile(actual, actualXDSStaticResourcesWithProbesFileName)

			Expect(actualYAML).To(Equal(expectedYAML),
				fmt.Sprintf("Compare files %s and %s\nExpected: %s\nActual struct: %s",
					expectedXDSStaticResourcesWithProbesFileName, actualXDSStaticResourcesWithProbesFileName, expectedYAML, actualYAML))
		})
	})

	Context("Test getEnvoyContainerPorts()", func() {
		It("creates container port list", func() {
			actualRewrittenContainerPorts := getEnvoyContainerPorts(originalHealthProbes)
			Expect(actualRewrittenContainerPorts).To(Equal(expectedRewrittenContainerPorts))
		})
	})

	Context("test getEnvoySidecarContainerSpec()", func() {
		It("creates Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)
			actual := getEnvoySidecarContainerSpec(pod, envoyImage, mockConfigurator, originalHealthProbes)

			expected := corev1.Container{
				Name:            constants.EnvoyContainerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullAlways,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: func() *int64 {
						uid := constants.EnvoyUID
						return &uid
					}(),
				},
				Ports: expectedRewrittenContainerPorts,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      envoyBootstrapConfigVolume,
						ReadOnly:  true,
						MountPath: envoyProxyConfigPath,
					},
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-node", "$(POD_UID)/$(POD_NAMESPACE)/$(POD_IP)/$(SERVICE_ACCOUNT)/svcacc/$(POD_NAME)/workload-kind/workload-name",
					"--service-cluster", "svcacc.namespace",
					"--bootstrap-version 3",
				},
				Env: []corev1.EnvVar{
					{
						Name:  "POD_UID",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.uid",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAME",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.name",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_NAMESPACE",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "metadata.namespace",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "POD_IP",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "status.podIP",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
					{
						Name:  "SERVICE_ACCOUNT",
						Value: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								APIVersion: "",
								FieldPath:  "spec.serviceAccountName",
							},
							ResourceFieldRef: nil,
							ConfigMapKeyRef:  nil,
							SecretKeyRef:     nil,
						},
					},
				},
			}

			Expect(actual).To(Equal(expected))
		})
	})
})

package injector

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mapset "github.com/deckarep/golang-set"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/version"
)

var _ = Describe("Test functions creating Envoy bootstrap configuration", func() {
	const (
		containerName = "-container-name-"
		envoyImage    = "-envoy-image-"
		clusterID     = "-cluster-id-"

		// This file contains the Bootstrap YAML generated for all of the Envoy proxies in OSM.
		// This is provisioned by the MutatingWebhook during the addition of a sidecar
		// to every new Pod that is being created in a namespace participating in the service mesh.
		// We deliberately leave this entire string literal here to document and visualize what the
		// generated YAML looks like!
		expectedEnvoyBootstrapConfigFileName        = "expected_envoy_bootstrap_config.yaml"
		actualGeneratedEnvoyBootstrapConfigFileName = "actual_envoy_bootstrap_config.yaml"

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

	meshConfig := v1alpha2.MeshConfig{
		Spec: v1alpha2.MeshConfigSpec{
			Sidecar: v1alpha2.SidecarSpec{
				TLSMinProtocolVersion: "TLSv1_2",
				TLSMaxProtocolVersion: "TLSv1_3",
				CipherSuites:          []string{},
			},
		},
	}

	cert := tresor.NewFakeCertificate()
	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetMeshConfig().Return(meshConfig).AnyTimes()

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
			log.Error().Err(err).Msgf("Error reading expected Envoy bootstrap YAML from file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
		return string(expectedEnvoyConfig)
	}

	saveActualEnvoyYAML := func(filename string, actualContent []byte) {
		err := ioutil.WriteFile(filepath.Clean(path.Join(directoryForYAMLFiles, filename)), actualContent, 0600)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing actual Envoy Cluster XDS YAML to file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
	}

	probes := healthProbes{
		liveness:  &healthProbe{path: "/liveness", port: 81, isHTTP: true},
		readiness: &healthProbe{path: "/readiness", port: 82, isHTTP: true},
		startup:   &healthProbe{path: "/startup", port: 83, isHTTP: true},
	}

	config := envoyBootstrapConfigMeta{
		NodeID:   cert.GetCommonName().String(),
		RootCert: cert.GetIssuingCA(),
		Cert:     cert.GetCertificateChain(),
		Key:      cert.GetPrivateKey(),

		EnvoyAdminPort: 15000,

		XDSClusterName: constants.OSMControllerName,
		XDSHost:        "osm-controller.b.svc.cluster.local",
		XDSPort:        15128,

		OriginalHealthProbes:  probes,
		TLSMinProtocolVersion: meshConfig.Spec.Sidecar.TLSMinProtocolVersion,
		TLSMaxProtocolVersion: meshConfig.Spec.Sidecar.TLSMaxProtocolVersion,
		CipherSuites:          meshConfig.Spec.Sidecar.CipherSuites,
		ECDHCurves:            meshConfig.Spec.Sidecar.ECDHCurves,
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
				configurator:        mockConfigurator,
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

		It("Updates bootstrap config for the Envoy proxy if it already exists", func() {
			name := uuid.New().String()
			namespace := "a"
			osmNamespace := "b"
			meta := metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels: map[string]string{
					constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
					constants.OSMAppInstanceLabelKey: "some-mesh",
					constants.OSMAppVersionLabelKey:  version.Version,
				},
			}
			existing := &corev1.Secret{
				ObjectMeta: meta,
				Data: map[string][]byte{
					"old": []byte("data"),
				},
			}
			wh := &mutatingWebhook{
				kubeClient:          fake.NewSimpleClientset(existing),
				kubeController:      k8s.NewMockController(gomock.NewController(GinkgoT())),
				nonInjectNamespaces: mapset.NewSet(),
				meshName:            "some-mesh",
				configurator:        mockConfigurator,
			}

			secret, err := wh.createEnvoyBootstrapConfig(name, namespace, osmNamespace, cert, probes)
			Expect(err).ToNot(HaveOccurred())

			expected := corev1.Secret{
				ObjectMeta: meta,
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

	Context("Test getProbeResources()", func() {
		It("Should not create listeners and clusters when there are no probes", func() {
			config.OriginalHealthProbes = healthProbes{} // no probes
			actualListeners, actualClusters, err := getProbeResources(config)
			Expect(err).To(BeNil())
			Expect(actualListeners).To(BeNil())
			Expect(actualClusters).To(BeNil())
		})

		It("Should not create listeners and cluster for TCPSocket probes", func() {
			config.OriginalHealthProbes = healthProbes{
				liveness:  &healthProbe{port: 81, isTCPSocket: true},
				readiness: &healthProbe{port: 82, isTCPSocket: true},
				startup:   &healthProbe{port: 83, isTCPSocket: true},
			}
			actualListeners, actualClusters, err := getProbeResources(config)
			Expect(err).To(BeNil())
			Expect(actualListeners).To(BeNil())
			Expect(actualClusters).To(BeNil())
		})
	})

	Context("Test getEnvoyContainerPorts()", func() {
		It("creates container port list", func() {
			actualRewrittenContainerPorts := getEnvoyContainerPorts(originalHealthProbes)
			Expect(actualRewrittenContainerPorts).To(Equal(expectedRewrittenContainerPorts))
		})
	})

	Context("test unix getEnvoySidecarContainerSpec()", func() {
		It("creates Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)
			mockConfigurator.EXPECT().GetEnvoyImage().Return(envoyImage).Times(1)
			mockConfigurator.EXPECT().GetEnvoyWindowsImage().Return(envoyImage).Times(0)
			mockConfigurator.EXPECT().GetProxyResources().Return(corev1.ResourceRequirements{
				// Test set Limits
				Limits: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("2"),
					"memory": resource.MustParse("512M"),
				},
				// Test unset Requests
				Requests: nil,
			}).Times(1)
			actual := getEnvoySidecarContainerSpec(pod, mockConfigurator, originalHealthProbes, constants.OSLinux)

			expected := corev1.Container{
				Name:            constants.EnvoyContainerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
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
				Resources: corev1.ResourceRequirements{
					// Test set Limits
					Limits: map[corev1.ResourceName]resource.Quantity{
						"cpu":    resource.MustParse("2"),
						"memory": resource.MustParse("512M"),
					},
					// Test unset Requests
					Requests: nil,
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-cluster", "svcacc.namespace",
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

	Context("test Windows getEnvoySidecarContainerSpec()", func() {
		It("creates Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)
			mockConfigurator.EXPECT().GetEnvoyWindowsImage().Return(envoyImage).Times(1)
			mockConfigurator.EXPECT().GetEnvoyImage().Return(envoyImage).Times(0)
			mockConfigurator.EXPECT().GetProxyResources().Return(corev1.ResourceRequirements{
				// Test set Limits
				Limits: map[corev1.ResourceName]resource.Quantity{
					"cpu":    resource.MustParse("2"),
					"memory": resource.MustParse("512M"),
				},
				// Test unset Requests
				Requests: nil,
			}).Times(1)
			actual := getEnvoySidecarContainerSpec(pod, mockConfigurator, originalHealthProbes, constants.OSWindows)

			expected := corev1.Container{
				Name:            constants.EnvoyContainerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					WindowsOptions: &corev1.WindowsSecurityContextOptions{
						RunAsUserName: func() *string {
							userName := "EnvoyUser"
							return &userName
						}(),
					},
				},
				Ports: expectedRewrittenContainerPorts,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      envoyBootstrapConfigVolume,
						ReadOnly:  true,
						MountPath: envoyProxyConfigPath,
					},
				},
				Resources: corev1.ResourceRequirements{
					// Test set Limits
					Limits: map[corev1.ResourceName]resource.Quantity{
						"cpu":    resource.MustParse("2"),
						"memory": resource.MustParse("512M"),
					},
					// Test unset Requests
					Requests: nil,
				},
				Command: []string{
					"envoy",
				},
				Args: []string{
					"--log-level", "debug",
					"--config-path", "/etc/envoy/bootstrap.yaml",
					"--service-cluster", "svcacc.namespace",
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

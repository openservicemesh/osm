package injector

import (
	"encoding/base64"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	k8s "github.com/openservicemesh/osm/pkg/kubernetes"
)

var _ = Describe("Test Envoy configuration creation", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	// This is the Bootstrap YAML generated for the Envoy proxies.
	// This is provisioned by the MutatingWebhook during the addition of a sidecar
	// to every new Pod that is being created in a namespace participating in the service mesh.
	// We deliberately leave this entire string literal here to document and visualize what the
	// generated YAML looks like!
	const expectedEnvoyConfig = `admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: "15000"
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: osm-controller
    set_node_on_first_message_only: true
    transport_api_version: V3
  cds_config:
    ads: {}
    resource_api_version: V3
  lds_config:
    ads: {}
    resource_api_version: V3
static_resources:
  clusters:
  - connect_timeout: 0.25s
    http2_protocol_options: {}
    load_assignment:
      cluster_name: osm-controller
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: osm-controller.b.svc.cluster.local
                port_value: 15128
    name: osm-controller
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        common_tls_context:
          alpn_protocols:
          - h2
          tls_certificates:
          - certificate_chain:
              inline_bytes: eHg=
            private_key:
              inline_bytes: eXk=
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
            tls_minimum_protocol_version: TLSv1_2
          validation_context:
            trusted_ca:
              inline_bytes: eHg=
    type: LOGICAL_DNS
`

	cert := tresor.NewFakeCertificate()

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("create envoy config", func() {
		It("creates envoy config", func() {
			config := envoyBootstrapConfigMeta{
				RootCert: base64.StdEncoding.EncodeToString(cert.GetIssuingCA()),
				Cert:     base64.StdEncoding.EncodeToString(cert.GetCertificateChain()),
				Key:      base64.StdEncoding.EncodeToString(cert.GetPrivateKey()),

				EnvoyAdminPort: 15000,

				XDSClusterName: "osm-controller",
				XDSHost:        "osm-controller.b.svc.cluster.local",
				XDSPort:        15128,
			}

			actual, err := getEnvoyConfigYAML(config, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(actual)).To(Equal(expectedEnvoyConfig),
				fmt.Sprintf("Expected:\n%s\nActual:\n%s\n", expectedEnvoyConfig, string(actual)))
		})

		It("Creates bootstrap config for the Envoy proxy", func() {
			wh := &webhook{
				kubeClient:     fake.NewSimpleClientset(),
				kubeController: k8s.NewMockController(gomock.NewController(GinkgoT())),
			}
			name := uuid.New().String()
			namespace := "a"
			osmNamespace := "b"

			secret, err := wh.createEnvoyBootstrapConfig(name, namespace, osmNamespace, cert)
			Expect(err).ToNot(HaveOccurred())

			expected := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					envoyBootstrapConfigFile: []byte(expectedEnvoyConfig),
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
})

var _ = Describe("Test Envoy sidecar", func() {
	const (
		containerName = "-container-name-"
		envoyImage    = "-envoy-image-"
		nodeID        = "-node-id-"
		clusterID     = "-cluster-id-"
	)

	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	Context("create Envoy sidecar", func() {
		It("creates correct Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)
			actual := getEnvoySidecarContainerSpec(containerName, envoyImage, nodeID, clusterID, mockConfigurator)
			Expect(len(actual)).To(Equal(1))

			expected := corev1.Container{
				Name:            containerName,
				Image:           envoyImage,
				ImagePullPolicy: corev1.PullAlways,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser: func() *int64 {
						uid := constants.EnvoyUID
						return &uid
					}(),
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          constants.EnvoyAdminPortName,
						ContainerPort: constants.EnvoyAdminPort,
					},
					{
						Name:          constants.EnvoyInboundListenerPortName,
						ContainerPort: constants.EnvoyInboundListenerPort,
					},
					{
						Name:          constants.EnvoyInboundPrometheusListenerPortName,
						ContainerPort: constants.EnvoyPrometheusInboundListenerPort,
					},
				},
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
					"--service-node", nodeID,
					"--service-cluster", clusterID,
					"--bootstrap-version 3",
				},
			}
			Expect(actual[0]).To(Equal(expected))
		})
	})
})

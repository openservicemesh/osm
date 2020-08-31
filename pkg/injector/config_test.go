package injector

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

const expectedEnvoyConfig = `
admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: "3465"
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: XDSClusterName
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
      cluster_name: XDSClusterName
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: XDSHost
                port_value: 2345
    name: XDSClusterName
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        common_tls_context:
          alpn_protocols:
          - h2
          tls_certificates:
          - certificate_chain:
              inline_bytes: Cert
            private_key:
              inline_bytes: Key
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
            tls_minimum_protocol_version: TLSv1_2
          validation_context:
            trusted_ca:
              inline_bytes: RootCert
    type: LOGICAL_DNS
`

var _ = Describe("Test Envoy configuration creation", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("create envoy config", func() {
		It("creates envoy config", func() {
			config := envoyBootstrapConfigMeta{
				EnvoyAdminPort: 3465,
				XDSClusterName: "XDSClusterName",
				RootCert:       "RootCert",
				Cert:           "Cert",
				Key:            "Key",
				XDSHost:        "XDSHost",
				XDSPort:        2345,
			}

			actual, err := getEnvoyConfigYAML(config, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(actual)).To(Equal(expectedEnvoyConfig[1:]),
				fmt.Sprintf("Expected:\n%s\nActual:\n%s\n", expectedEnvoyConfig, string(actual)))
		})
	})
})

var _ = Describe("Test Envoy sidecar", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("create Envoy sidecar", func() {
		It("creates correct Envoy sidecar spec", func() {
			mockConfigurator.EXPECT().GetEnvoyLogLevel().Return("debug").Times(1)

			actual := getEnvoySidecarContainerSpec("a", "b", "c", "d", mockConfigurator)
			Expect(len(actual)).To(Equal(1))

			expected := corev1.Container{
				Name:            "a",
				Image:           "b",
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
					"--service-node", "c",
					"--service-cluster", "d",
					"--bootstrap-version 3",
				},
			}
			Expect(actual[0]).To(Equal(expected))
		})
	})
})

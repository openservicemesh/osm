package injector

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
  cds_config:
    ads: {}
  lds_config:
    ads: {}
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
    tls_context:
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
tracing:
  http:
    name: envoy.zipkin
    typed_config:
      '@type': type.googleapis.com/envoy.config.trace.v2.ZipkinConfig
      collector_cluster: envoy-zipkin-cluster
      collector_endpoint: /api/v2/spans
      collector_endpoint_version: HTTP_JSON
`

var _ = Describe("Test Envoy configuration creation", func() {
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

			actual, err := getEnvoyConfigYAML(config)
			Expect(err).ToNot(HaveOccurred())

			Expect(string(actual)).To(Equal(expectedEnvoyConfig[1:]),
				fmt.Sprintf("Expected:\n%s\nActual:\n%s\n", expectedEnvoyConfig, string(actual)))
		})
	})
})

var _ = Describe("Test Envoy config path creation function", func() {
	Context("create envoy config path", func() {
		It("creates envoy config path", func() {
			Expect(getEnvoyConfigPath()).To(Equal("/etc/envoy/bootstrap.yaml"))
		})
	})
})

package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const envoyBootstrapConfigTmpl = `
admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      address: 0.0.0.0
      port_value: '{{.EnvoyAdminPort}}'
dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: '{{.XDSClusterName}}'
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
      cluster_name: '{{.XDSClusterName}}'
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: '{{.XDSHost}}'
                port_value: '{{.XDSPort}}'
    name: '{{.XDSClusterName}}'
    tls_context:
      common_tls_context:
        alpn_protocols:
        - h2
        tls_certificates:
        - certificate_chain:
            inline_bytes: '{{.Cert}}'
          private_key:
            inline_bytes: '{{.Key}}'
        tls_params:
          cipher_suites: '[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]'
          tls_maximum_protocol_version: TLSv1_3
          tls_minimum_protocol_version: TLSv1_2
        validation_context:
          trusted_ca:
            inline_bytes: '{{.RootCert}}'
    type: LOGICAL_DNS
`

var _ = Describe("Test Envoy configuration creation", func() {
	Context("create envoy config", func() {
		It("creates envoy config", func() {
			actual := getEnvoyConfigYAML()
			expected := envoyBootstrapConfigTmpl[1:]
			Expect(actual).To(Equal(expected))
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

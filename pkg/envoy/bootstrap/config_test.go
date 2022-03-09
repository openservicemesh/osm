package bootstrap

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/utils"
)

func TestBuildFromConfig(t *testing.T) {
	assert := tassert.New(t)
	cert := tresor.NewFakeCertificate()

	config := Config{
		NodeID:                cert.GetCommonName().String(),
		AdminPort:             15000,
		XDSClusterName:        constants.OSMControllerName,
		TrustedCA:             cert.GetIssuingCA(),
		CertificateChain:      cert.GetCertificateChain(),
		PrivateKey:            cert.GetPrivateKey(),
		XDSHost:               "osm-controller.osm-system.svc.cluster.local",
		XDSPort:               15128,
		TLSMinProtocolVersion: "TLSv1_0",
		TLSMaxProtocolVersion: "TLSv1_2",
		CipherSuites:          []string{"abc", "xyz"},
		ECDHCurves:            []string{"ABC", "XYZ"},
	}

	bootstrapConfig, err := BuildFromConfig(config)
	assert.Nil(err)
	assert.NotNil(bootstrapConfig)

	actualYAML, err := utils.ProtoToYAML(bootstrapConfig)
	assert.Nil(err)

	expectedYAML := `admin:
  access_log:
  - name: envoy.access_loggers.stream
    typed_config:
      '@type': type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 15000
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
node:
  id: foo.bar.co.uk
static_resources:
  clusters:
  - load_assignment:
      cluster_name: osm-controller
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: osm-controller.osm-system.svc.cluster.local
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
            cipher_suites:
            - abc
            - xyz
            ecdh_curves:
            - ABC
            - XYZ
            tls_maximum_protocol_version: TLSv1_2
            tls_minimum_protocol_version: TLSv1_0
          validation_context:
            trusted_ca:
              inline_bytes: eHg=
    type: LOGICAL_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
`
	assert.Equal(expectedYAML, string(actualYAML))
}

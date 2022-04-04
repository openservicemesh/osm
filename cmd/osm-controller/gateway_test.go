package main

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
)

func TestBootstrapOSMMulticlusterGateway(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	fakeCertManager := tresor.NewFake(nil)

	testCases := []struct {
		name            string
		bootstrapSecret *corev1.Secret
		expectError     bool
	}{
		{
			name:            "secret does not exist, it should be created by Helm",
			bootstrapSecret: nil,
			expectError:     true,
		},
		{
			name: "Secret with placeholder config exists",
			bootstrapSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatewayBootstrapSecretName,
					Namespace: osmNamespace,
				},
				Data: map[string][]byte{
					bootstrapConfigKey: []byte("-- placeholder --"),
				},
			},
			expectError: false,
		},
		{
			name: "Secret with valid config exists",
			bootstrapSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatewayBootstrapSecretName,
					Namespace: osmNamespace,
				},
				Data: map[string][]byte{
					bootstrapConfigKey: []byte(`admin:
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
  id: ab394bf2-3a1e-4f59-a182-688a732f180f.gateway.osm.osm-system
static_resources:
  clusters:
  - connect_timeout: 0.250s
    load_assignment:
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
              inline_bytes: --redacted--
            private_key:
              inline_bytes: --redacted--
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
            tls_minimum_protocol_version: TLSv1_2
          validation_context:
            trusted_ca:
              inline_bytes: --redacted--
    type: LOGICAL_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
`),
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			fakeClient := fake.NewSimpleClientset()

			testNs := "test"
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNs,
				},
			}

			_, err := fakeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			assert.Nil(err)

			if tc.bootstrapSecret != nil {
				_, err := fakeClient.CoreV1().Secrets(testNs).Create(context.Background(), tc.bootstrapSecret, metav1.CreateOptions{})
				assert.Nil(err)
			}

			actual := bootstrapOSMMulticlusterGateway(fakeClient, fakeCertManager, testNs)
			assert.Equal(tc.expectError, actual != nil)
		})
	}
}

func TestIsValidBootstrapData(t *testing.T) {
	testCases := []struct {
		name         string
		boostrapYAML string
		expected     bool
	}{
		{
			name: "valid bootstrap config",
			boostrapYAML: `admin:
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
  id: ab394bf2-3a1e-4f59-a182-688a732f180f.gateway.osm.osm-system
static_resources:
  clusters:
  - connect_timeout: 0.250s
    load_assignment:
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
              inline_bytes: --redacted--
            private_key:
              inline_bytes: --redacted--
          tls_params:
            tls_maximum_protocol_version: TLSv1_3
            tls_minimum_protocol_version: TLSv1_2
          validation_context:
            trusted_ca:
              inline_bytes: --redacted--
    type: LOGICAL_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
`,
			expected: true,
		},
		{
			name: "invalid bootstrap config, missing static_resources",
			boostrapYAML: `admin:
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
  id: ab394bf2-3a1e-4f59-a182-688a732f180f.gateway.osm.osm-system
`,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			actual := isValidBootstrapData([]byte(tc.boostrapYAML))
			assert.Equal(tc.expected, actual)
		})
	}
}

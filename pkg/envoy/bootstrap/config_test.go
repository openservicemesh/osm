package bootstrap

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

func TestBuild(t *testing.T) {
	assert := tassert.New(t)
	cert := tresorFake.NewFakeCertificate()

	b := &Builder{
		NodeID:                cert.GetCommonName().String(),
		XDSHost:               "osm-controller.osm-system.svc.cluster.local",
		TLSMinProtocolVersion: "TLSv1_0",
		TLSMaxProtocolVersion: "TLSv1_2",
		CipherSuites:          []string{"abc", "xyz"},
		ECDHCurves:            []string{"ABC", "XYZ"},
		OriginalHealthProbes: models.HealthProbes{
			Liveness:  &models.HealthProbe{Path: "/liveness", Port: 81, IsHTTP: true},
			Readiness: &models.HealthProbe{Path: "/readiness", Port: 82, IsHTTP: true},
			Startup:   &models.HealthProbe{Path: "/startup", Port: 83, IsHTTP: true},
		},
	}

	bootstrapConfig, err := b.Build()
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
          tls_certificate_sds_secret_configs:
          - name: tls_sds
            sds_config:
              path: /etc/envoy/tls_certificate_sds_secret.yaml
          tls_params:
            cipher_suites:
            - abc
            - xyz
            ecdh_curves:
            - ABC
            - XYZ
            tls_maximum_protocol_version: TLSv1_2
            tls_minimum_protocol_version: TLSv1_0
          validation_context_sds_secret_config:
            name: validation_context_sds
            sds_config:
              path: /etc/envoy/validation_context_sds_secret.yaml
    type: LOGICAL_DNS
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        '@type': type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
  - load_assignment:
      cluster_name: liveness_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 81
    name: liveness_cluster
    type: STATIC
  - load_assignment:
      cluster_name: readiness_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 82
    name: readiness_cluster
    type: STATIC
  - load_assignment:
      cluster_name: startup_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1
                port_value: 83
    name: startup_cluster
    type: STATIC
  listeners:
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15901
    filter_chains:
    - filters:
      - name: http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          access_log:
          - name: envoy.access_loggers.stream
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
              log_format:
                json_format:
                  authority: '%REQ(:AUTHORITY)%'
                  bytes_received: '%BYTES_RECEIVED%'
                  bytes_sent: '%BYTES_SENT%'
                  duration: '%DURATION%'
                  method: '%REQ(:METHOD)%'
                  path: '%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%'
                  protocol: '%PROTOCOL%'
                  request_id: '%REQ(X-REQUEST-ID)%'
                  requested_server_name: '%REQUESTED_SERVER_NAME%'
                  response_code: '%RESPONSE_CODE%'
                  response_code_details: '%RESPONSE_CODE_DETAILS%'
                  response_flags: '%RESPONSE_FLAGS%'
                  start_time: '%START_TIME%'
                  time_to_first_byte: '%RESPONSE_DURATION%'
                  upstream_cluster: '%UPSTREAM_CLUSTER%'
                  upstream_host: '%UPSTREAM_HOST%'
                  upstream_service_time: '%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%'
                  user_agent: '%REQ(USER-AGENT)%'
                  x_forwarded_for: '%REQ(X-FORWARDED-FOR)%'
          http_filters:
          - name: http_router
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-liveness-probe
                route:
                  cluster: liveness_cluster
                  prefix_rewrite: /liveness
                  timeout: 1s
          stat_prefix: health_probes_http
    name: liveness_listener
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15902
    filter_chains:
    - filters:
      - name: http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          access_log:
          - name: envoy.access_loggers.stream
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
              log_format:
                json_format:
                  authority: '%REQ(:AUTHORITY)%'
                  bytes_received: '%BYTES_RECEIVED%'
                  bytes_sent: '%BYTES_SENT%'
                  duration: '%DURATION%'
                  method: '%REQ(:METHOD)%'
                  path: '%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%'
                  protocol: '%PROTOCOL%'
                  request_id: '%REQ(X-REQUEST-ID)%'
                  requested_server_name: '%REQUESTED_SERVER_NAME%'
                  response_code: '%RESPONSE_CODE%'
                  response_code_details: '%RESPONSE_CODE_DETAILS%'
                  response_flags: '%RESPONSE_FLAGS%'
                  start_time: '%START_TIME%'
                  time_to_first_byte: '%RESPONSE_DURATION%'
                  upstream_cluster: '%UPSTREAM_CLUSTER%'
                  upstream_host: '%UPSTREAM_HOST%'
                  upstream_service_time: '%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%'
                  user_agent: '%REQ(USER-AGENT)%'
                  x_forwarded_for: '%REQ(X-FORWARDED-FOR)%'
          http_filters:
          - name: http_router
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-readiness-probe
                route:
                  cluster: readiness_cluster
                  prefix_rewrite: /readiness
                  timeout: 1s
          stat_prefix: health_probes_http
    name: readiness_listener
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 15903
    filter_chains:
    - filters:
      - name: http_connection_manager
        typed_config:
          '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          access_log:
          - name: envoy.access_loggers.stream
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
              log_format:
                json_format:
                  authority: '%REQ(:AUTHORITY)%'
                  bytes_received: '%BYTES_RECEIVED%'
                  bytes_sent: '%BYTES_SENT%'
                  duration: '%DURATION%'
                  method: '%REQ(:METHOD)%'
                  path: '%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%'
                  protocol: '%PROTOCOL%'
                  request_id: '%REQ(X-REQUEST-ID)%'
                  requested_server_name: '%REQUESTED_SERVER_NAME%'
                  response_code: '%RESPONSE_CODE%'
                  response_code_details: '%RESPONSE_CODE_DETAILS%'
                  response_flags: '%RESPONSE_FLAGS%'
                  start_time: '%START_TIME%'
                  time_to_first_byte: '%RESPONSE_DURATION%'
                  upstream_cluster: '%UPSTREAM_CLUSTER%'
                  upstream_host: '%UPSTREAM_HOST%'
                  upstream_service_time: '%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%'
                  user_agent: '%REQ(USER-AGENT)%'
                  x_forwarded_for: '%REQ(X-FORWARDED-FOR)%'
          http_filters:
          - name: http_router
            typed_config:
              '@type': type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
          route_config:
            name: local_route
            virtual_hosts:
            - domains:
              - '*'
              name: local_service
              routes:
              - match:
                  prefix: /osm-startup-probe
                route:
                  cluster: startup_cluster
                  prefix_rewrite: /startup
                  timeout: 1s
          stat_prefix: health_probes_http
    name: startup_listener
`
	assert.Equal(expectedYAML, string(actualYAML))
}

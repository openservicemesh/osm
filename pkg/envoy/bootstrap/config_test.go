package bootstrap

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"

	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	"testing"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/configurator"

	tassert "github.com/stretchr/testify/assert"

	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
)

func TestBuildFromConfig(t *testing.T) {
	assert := tassert.New(t)
	cert := tresorFake.NewFakeCertificate()

	config := Config{
		NodeID:                cert.GetCommonName().String(),
		AdminPort:             15000,
		XDSClusterName:        constants.OSMControllerName,
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
`
	assert.Equal(expectedYAML, string(actualYAML))
}

var _ = Describe("Test functions creating Envoy bootstrap configuration", func() {
	const (
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

	meshConfig := v1alpha2.MeshConfig{
		Spec: v1alpha2.MeshConfigSpec{
			Sidecar: v1alpha2.SidecarSpec{
				TLSMinProtocolVersion: "TLSv1_2",
				TLSMaxProtocolVersion: "TLSv1_3",
				CipherSuites:          []string{},
			},
		},
	}

	cert := tresorFake.NewFakeCertificate()
	mockCtrl := gomock.NewController(GinkgoT())
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
	mockConfigurator.EXPECT().GetMeshConfig().Return(meshConfig).AnyTimes()

	getExpectedEnvoyYAML := func(filename string) string {
		expectedEnvoyConfig, err := ioutil.ReadFile(filepath.Clean(path.Join(directoryForYAMLFiles, filename)))
		if err != nil {
			log.Error().Err(err).Msgf("Error reading expected Envoy bootstrap YAML from file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
		return string(expectedEnvoyConfig)
	}

	getExpectedEnvoyConfig := func(filename string) *xds_bootstrap.Bootstrap {
		yaml := getExpectedEnvoyYAML(filename)
		conf := xds_bootstrap.Bootstrap{}
		err := utils.YAMLToProto([]byte(yaml), &conf)
		Expect(err).ToNot(HaveOccurred())
		return &conf
	}

	saveActualEnvoyConfig := func(filename string, actual *xds_bootstrap.Bootstrap) {
		actualContent, err := utils.ProtoToYAML(actual)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(filepath.Clean(path.Join(directoryForYAMLFiles, filename)), actualContent, 0600)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing actual Envoy Cluster XDS YAML to file %s", filename)
		}
		Expect(err).ToNot(HaveOccurred())
	}

	probes := models.HealthProbes{
		Liveness:  &models.HealthProbe{Path: "/liveness", Port: 81, IsHTTP: true},
		Readiness: &models.HealthProbe{Path: "/readiness", Port: 82, IsHTTP: true},
		Startup:   &models.HealthProbe{Path: "/startup", Port: 83, IsHTTP: true},
	}

	config := EnvoyBootstrapConfigMeta{
		NodeID: cert.GetCommonName().String(),

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

	Context("Test GenerateEnvoyConfig()", func() {
		It("creates Envoy bootstrap config", func() {
			config.OriginalHealthProbes = probes
			actual, err := GenerateEnvoyConfig(config, mockConfigurator)
			Expect(err).ToNot(HaveOccurred())
			saveActualEnvoyConfig(actualGeneratedEnvoyBootstrapConfigFileName, actual)

			expectedEnvoyConfig := getExpectedEnvoyConfig(expectedEnvoyBootstrapConfigFileName)

			actualYaml, err := utils.ProtoToYAML(actual)
			Expect(err).ToNot(HaveOccurred())

			expectedYaml, err := utils.ProtoToYAML(expectedEnvoyConfig)
			Expect(err).ToNot(HaveOccurred())

			Expect(actualYaml).To(Equal(expectedYaml),
				fmt.Sprintf("	 %s and %s\nExpected:\n%s\nActual:\n%s\n",
					expectedEnvoyBootstrapConfigFileName, actualGeneratedEnvoyBootstrapConfigFileName, expectedYaml, actualYaml))
		})
	})

	Context("Test getProbeResources()", func() {
		It("Should not create listeners and clusters when there are no probes", func() {
			config.OriginalHealthProbes = models.HealthProbes{} // no probes
			actualListeners, actualClusters, err := getProbeResources(config)
			Expect(err).To(BeNil())
			Expect(actualListeners).To(BeNil())
			Expect(actualClusters).To(BeNil())
		})

		It("Should not create listeners and cluster for TCPSocket probes", func() {
			config.OriginalHealthProbes = models.HealthProbes{
				Liveness:  &models.HealthProbe{Port: 81, IsTCPSocket: true},
				Readiness: &models.HealthProbe{Port: 82, IsTCPSocket: true},
				Startup:   &models.HealthProbe{Port: 83, IsTCPSocket: true},
			}
			actualListeners, actualClusters, err := getProbeResources(config)
			Expect(err).To(BeNil())
			Expect(actualListeners).To(BeNil())
			Expect(actualClusters).To(BeNil())
		})
	})
})

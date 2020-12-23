package injector

import (
	"encoding/base64"
	"fmt"
	"strconv"

	ghyaml "github.com/ghodss/yaml"
	"gopkg.in/yaml.v2"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
)

func getEnvoyConfigYAML(config envoyBootstrapConfigMeta) ([]byte, []byte, error) {
	m := map[interface{}]interface{}{
		"admin": map[string]interface{}{
			"access_log_path": "/dev/stdout",
			"address": map[string]interface{}{
				"socket_address": map[string]string{
					"address":    "0.0.0.0",
					"port_value": strconv.Itoa(config.EnvoyAdminPort),
				},
			},
		},

		"dynamic_resources": map[string]interface{}{
			"ads_config": map[string]interface{}{
				"api_type":              "GRPC",
				"transport_api_version": "V3",
				"grpc_services": []map[string]interface{}{
					{
						"envoy_grpc": map[string]interface{}{
							"cluster_name": config.XDSClusterName,
						},
					},
				},
				"set_node_on_first_message_only": true,
			},
			"cds_config": map[string]interface{}{
				"ads":                  map[string]string{},
				"resource_api_version": "V3",
			},
			"lds_config": map[string]interface{}{
				"ads":                  map[string]string{},
				"resource_api_version": "V3",
			},
		},
	}

	m["static_resources"] = getStaticResources(config)

	configYAML, err := yaml.Marshal(&m)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling Envoy config struct into YAML")
		return nil, nil, err
	}

	configJSON, err := ghyaml.YAMLToJSON(configYAML)
	if err != nil {
		log.Error().Err(err).Msgf("Error converting YAML to JSON")
		return nil, nil, err
	}

	return configYAML, configJSON, err
}

func getStaticResources(config envoyBootstrapConfigMeta) map[string]interface{} {
	clusters := []map[string]interface{}{
		getXdsCluster(config),
	}

	staticResources := map[string]interface{}{
		"clusters": clusters,
	}

	return staticResources
}

func getEnvoyBootstrapConfig(osmNamespace string, cert certificate.Certificater) ([]byte, error) {
	configMeta := envoyBootstrapConfigMeta{
		EnvoyAdminPort: constants.EnvoyAdminPort,
		XDSClusterName: constants.OSMControllerName,

		RootCert: base64.StdEncoding.EncodeToString(cert.GetIssuingCA()),
		Cert:     base64.StdEncoding.EncodeToString(cert.GetCertificateChain()),
		Key:      base64.StdEncoding.EncodeToString(cert.GetPrivateKey()),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.OSMControllerName, osmNamespace),
		XDSPort: constants.OSMControllerPort,
	}
	_, jsonConfig, err := getEnvoyConfigYAML(configMeta)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy bootstrap YAML")
		return nil, err
	}

	return jsonConfig, err
}

func getXdsCluster(config envoyBootstrapConfigMeta) map[string]interface{} {
	return map[string]interface{}{
		"name":                   config.XDSClusterName,
		"connect_timeout":        "0.25s",
		"type":                   "LOGICAL_DNS",
		"http2_protocol_options": map[string]string{},
		"transport_socket": map[string]interface{}{
			"name": "envoy.transport_sockets.tls",
			"typed_config": map[string]interface{}{
				"@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
				"common_tls_context": map[string]interface{}{
					"alpn_protocols": []string{
						"h2",
					},
					"validation_context": map[string]interface{}{
						"trusted_ca": map[string]interface{}{
							"inline_bytes": config.RootCert,
						},
					},
					"tls_params": map[string]interface{}{
						"tls_minimum_protocol_version": "TLSv1_2",
						"tls_maximum_protocol_version": "TLSv1_3",
					},
					"tls_certificates": []map[string]interface{}{
						{
							"certificate_chain": map[string]interface{}{
								"inline_bytes": config.Cert,
							},
							"private_key": map[string]interface{}{
								"inline_bytes": config.Key,
							},
						},
					},
				},
			},
		},
		"load_assignment": map[string]interface{}{
			"cluster_name": config.XDSClusterName,
			"endpoints": []map[string]interface{}{
				{
					"lb_endpoints": []map[string]interface{}{
						{
							"endpoint": map[string]interface{}{
								"address": map[string]interface{}{
									"socket_address": map[string]interface{}{
										"address":    config.XDSHost,
										"port_value": config.XDSPort,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

package injector

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

func getEnvoyConfigYAML(config envoyBootstrapConfigMeta, cfg configurator.Configurator) ([]byte, error) {
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

		"static_resources": map[string]interface{}{
			"clusters": []map[string]interface{}{
				{
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
				},
			},
		},
	}

	configYAML, err := yaml.Marshal(&m)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling Envoy config struct into YAML")
		return nil, err
	}
	return configYAML, err
}

func (wh *webhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string, cert certificate.Certificater) (*corev1.Secret, error) {
	configMeta := envoyBootstrapConfigMeta{
		EnvoyAdminPort: constants.EnvoyAdminPort,
		XDSClusterName: constants.OSMControllerName,

		RootCert: base64.StdEncoding.EncodeToString(cert.GetIssuingCA()),
		Cert:     base64.StdEncoding.EncodeToString(cert.GetCertificateChain()),
		Key:      base64.StdEncoding.EncodeToString(cert.GetPrivateKey()),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.OSMControllerName, osmNamespace),
		XDSPort: constants.OSMControllerPort,
	}
	yamlContent, err := getEnvoyConfigYAML(configMeta, wh.configurator)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy bootstrap YAML")
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			envoyBootstrapConfigFile: yamlContent,
		},
	}
	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{}); err == nil {
		log.Info().Msgf("Updating bootstrap config Envoy: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	}

	log.Info().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}

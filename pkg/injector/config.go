package injector

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/open-service-mesh/osm/pkg/envoy"
)

func getEnvoyConfigYAML(config envoyBootstrapConfigMeta) ([]byte, error) {
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

		"tracing": map[string]interface{}{
			"http": map[string]interface{}{
				"name": "envoy.zipkin",
				"typed_config": map[string]interface{}{
					"@type":                      envoy.TypeZipkinConfig,
					"collector_cluster":          constants.EnvoyZipkinCluster,
					"collector_endpoint":         constants.EnvoyZipkinEndpoint,
					"collector_endpoint_version": "HTTP_JSON",
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
	yamlContent, err := getEnvoyConfigYAML(configMeta)
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy bootstrap YAML")
		return nil, err
	}

	if len(wh.config.StatsWASMExtensionPath) > 0 {
		_, err := createOrUpdateWasmConfigMap(wh, namespace, osmNamespace)
		if err != nil {
			log.Error().Msgf("Error Creating/updating config map : %s", err)
		}
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

func createOrUpdateWasmConfigMap(wh *webhook, namespace, osmNamespace string) (*corev1.ConfigMap, error) {

	// Read WASM module to attach.
	statsWASM, err := ioutil.ReadFile(wh.config.StatsWASMExtensionPath)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Creating Configmap to hold WASM module")
	wasmConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: envoyWasmConfigMapName,
		},
		BinaryData: make(map[string][]byte),
	}
	wasmConfigMap.BinaryData[wh.config.StatsWASMExtensionPath] = statsWASM

	ret, err := wh.kubeClient.CoreV1().ConfigMaps(namespace).Get(context.Background(), envoyWasmConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ret, err = wh.kubeClient.CoreV1().ConfigMaps(namespace).Create(context.Background(), wasmConfigMap, metav1.CreateOptions{})
	} else {
		ret, err = wh.kubeClient.CoreV1().ConfigMaps(namespace).Update(context.Background(), wasmConfigMap, metav1.UpdateOptions{})
	}
	return ret, err
}

func getEnvoyConfigPath() string {
	return strings.Join([]string{envoyProxyConfigPath, envoyBootstrapConfigFile}, "/")
}

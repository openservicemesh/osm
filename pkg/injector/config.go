package injector

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"text/template"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
)

const bootstrapFile = "bootstrap.yaml"

const (
	tlsRootCertFileKey = "root-cert.pem"

	tlsCertFileKey = "cert.pem"

	tlsKeyFileKey = "key.pem"
)

func getEnvoyConfigYAML() string {
	m := map[interface{}]interface{}{
		"admin": map[string]interface{}{
			"access_log_path": "/dev/stdout",
			"address": map[string]interface{}{
				"socket_address": map[string]string{
					"address":    "0.0.0.0",
					"port_value": "{{.EnvoyAdminPort}}",
				},
			},
		},

		"dynamic_resources": map[string]interface{}{
			"ads_config": map[string]interface{}{
				"api_type": "GRPC",
				"grpc_services": []map[string]interface{}{
					{
						"envoy_grpc": map[string]interface{}{
							"cluster_name": "{{.XDSClusterName}}",
						},
					},
				},
				"set_node_on_first_message_only": true,
			},
			"cds_config": map[string]interface{}{
				"ads": map[string]string{},
			},
			"lds_config": map[string]interface{}{
				"ads": map[string]string{},
			},
		},

		"static_resources": map[string]interface{}{
			"clusters": []map[string]interface{}{
				{
					"name":                   "{{.XDSClusterName}}",
					"connect_timeout":        "0.25s",
					"type":                   "LOGICAL_DNS",
					"http2_protocol_options": map[string]string{},
					"tls_context": map[string]interface{}{
						"common_tls_context": map[string]interface{}{
							"alpn_protocols": []string{
								"h2",
							},
							"validation_context": map[string]interface{}{
								"trusted_ca": map[string]interface{}{
									"filename": "{{.RootCertPath}}",
								},
							},
							"tls_params": map[string]interface{}{
								"tls_minimum_protocol_version": "TLSv1_2",
								"tls_maximum_protocol_version": "TLSv1_3",
								"cipher_suites":                "[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]",
							},
							"tls_certificates": []map[string]interface{}{
								{
									"certificate_chain": map[string]interface{}{
										"filename": "{{.CertPath}}",
									},
									"private_key": map[string]interface{}{
										"filename": "{{.KeyPath}}",
									},
								},
							},
						},
					},
					"load_assignment": map[string]interface{}{
						"cluster_name": "{{.XDSClusterName}}",
						"endpoints": []map[string]interface{}{
							{
								"lb_endpoints": []map[string]interface{}{
									{
										"endpoint": map[string]interface{}{
											"address": map[string]interface{}{
												"socket_address": map[string]interface{}{
													"address":    "{{.XDSHost}}",
													"port_value": "{{.XDSPort}}",
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

	d, err := yaml.Marshal(&m)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error marshaling")
	}
	return string(d)
}

type envoyBootstrapConfigMeta struct {
	EnvoyAdminPort int32
	XDSClusterName string
	RootCertPath   string
	CertPath       string
	KeyPath        string
	XDSHost        string
	XDSPort        int32
}

func (wh *Webhook) createEnvoyTLSSecret(name string, namespace string, cert certificate.Certificater) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			tlsRootCertFileKey: cert.GetIssuingCA(),
			tlsCertFileKey:     cert.GetCertificateChain(),
			tlsKeyFileKey:      cert.GetPrivateKey(),
		},
	}

	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{}); err == nil {
		log.Info().Msgf("Updating secret for envoy certs: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(existing)
	}

	log.Info().Msgf("Creating secret for envoy certs: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(secret)
}

func (wh *Webhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string) (*corev1.Secret, error) {
	configMeta := envoyBootstrapConfigMeta{
		EnvoyAdminPort: constants.EnvoyAdminPort,
		XDSClusterName: constants.AggregatedDiscoveryServiceName,

		RootCertPath: path.Join(envoyCertificatesDirectory, tlsRootCertFileKey),
		CertPath:     path.Join(envoyCertificatesDirectory, tlsCertFileKey),
		KeyPath:      path.Join(envoyCertificatesDirectory, tlsKeyFileKey),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.AggregatedDiscoveryServiceName, osmNamespace),
		XDSPort: constants.AggregatedDiscoveryServicePort,
	}
	yamlContent, err := renderEnvoyBootstrapConfig(configMeta)
	if err != nil {
		log.Error().Err(err).Msg("Failed to render Envoy bootstrap config from template")
		return nil, err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			bootstrapFile: yamlContent,
		},
	}
	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{}); err == nil {
		log.Info().Msgf("Updating bootstrap config Envoy: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(existing)
	}

	log.Info().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(secret)
}

func renderEnvoyBootstrapConfig(configMeta envoyBootstrapConfigMeta) ([]byte, error) {
	tmpl, err := template.New("envoy-bootstrap-config").Parse(getEnvoyConfigYAML())
	if err != nil {
		return nil, err
	}

	var data bytes.Buffer
	w := bufio.NewWriter(&data)
	if err := tmpl.Execute(w, configMeta); err != nil {
		return nil, err
	}
	err = w.Flush()
	if err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

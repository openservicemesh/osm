package injector

import (
	"bufio"
	"bytes"
	"fmt"
	"path"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/open-service-mesh/osm/pkg/constants"
)

const (
	tlsRootCertFileKey = "root-cert.pem"
	tlsCertFileKey     = "cert.pem"
	tlsKeyFileKey      = "key.pem"
)

const envoyBootstrapConfigTmpl = `
admin:
  access_log_path: "/dev/stdout"
  address:
    socket_address: {address: 0.0.0.0, port_value: {{.EnvoyAdminPort}}}

dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: {{.XDSClusterName}}
    set_node_on_first_message_only: true
  cds_config:
    ads: {}
  lds_config:
    ads: {}

static_resources:
  clusters:

  - name: {{.XDSClusterName}}
    connect_timeout: 0.25s
    type: LOGICAL_DNS
    http2_protocol_options: {}
    tls_context:
      common_tls_context:
        alpn_protocols:
          - h2
        validation_context:
          trusted_ca: { filename: "{{.RootCertPath}}" }
        tls_params:
          tls_minimum_protocol_version: TLSv1_2
          tls_maximum_protocol_version: TLSv1_3
          cipher_suites: "[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]"
        tls_certificates:
          - certificate_chain: { filename: "{{.CertPath}}" }
            private_key: { filename: "{{.KeyPath}}" }
    load_assignment:
      cluster_name: {{.XDSClusterName}}
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: {{.XDSHost}}
                port_value: {{.XDSPort}}
`

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

	log.Info().Msgf(">>> Cert %+v", secret)

	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{}); err == nil {
		log.Info().Msgf("Updating secret for envoy certs: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(existing)
	}

	log.Info().Msgf("Creating secret for envoy certs: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(secret)
}

func (wh *Webhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string) (*corev1.ConfigMap, error) {
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

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"bootstrap.yaml": yamlContent,
		},
		BinaryData: nil,
	}

	if existing, err := wh.kubeClient.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{}); err == nil {
		log.Info().Msgf("Updating configMap for envoy bootstrap config: name=%s, namespace=%s", name, namespace)
		existing.Data = configMap.Data
		return wh.kubeClient.CoreV1().ConfigMaps(namespace).Update(existing)
	}

	log.Info().Msgf("Creating configMap for envoy bootstrap config: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().ConfigMaps(namespace).Create(configMap)
}

func renderEnvoyBootstrapConfig(configMeta envoyBootstrapConfigMeta) (string, error) {
	tmpl, err := template.New("envoy-bootstrap-config").Parse(envoyBootstrapConfigTmpl)
	if err != nil {
		return "", err
	}

	var data bytes.Buffer
	w := bufio.NewWriter(&data)
	if err := tmpl.Execute(w, configMeta); err != nil {
		return "", err
	}
	err = w.Flush()
	if err != nil {
		return "", err
	}
	return data.String(), nil
}

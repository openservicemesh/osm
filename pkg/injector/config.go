package injector

import (
	"bytes"
	"encoding/pem"
	"fmt"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-service-mesh/osm/demo/cmd/common"
	"github.com/open-service-mesh/osm/pkg/certificate"
	"github.com/pkg/errors"
)

const (
	tlsRootCertFileKey = "root-cert.pem"
	tlsCertFileKey     = "cert.pem"
	tlsKeyFileKey      = "key.pem"
)

func (wh *Webhook) createEnvoyTLSSecret(name string, namespace string, cert certificate.Certificater) (*corev1.Secret, error) {
	// PEM encode the root certificate
	block := pem.Block{Type: "CERTIFICATE", Bytes: cert.GetRootCertificate().Raw}
	var rootCert bytes.Buffer
	err := pem.Encode(&rootCert, &block)
	if err != nil {
		return nil, errors.Wrap(err, "error PEM encoding certificate")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			tlsRootCertFileKey: rootCert.Bytes(),
			tlsCertFileKey:     cert.GetCertificateChain(),
			tlsKeyFileKey:      cert.GetPrivateKey(),
		},
	}

	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{}); err == nil {
		glog.Infof("Updating secret for envoy certs: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(existing)
	}

	glog.Infof("Creating secret for envoy certs: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(secret)
}

func (wh *Webhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string) (*corev1.ConfigMap, error) {
	xdsHost := fmt.Sprintf("ads.%s.svc.cluster.local", osmNamespace)
	yamlContent := fmt.Sprintf(`---
admin:
  access_log_path: "/dev/stdout"
  address:
    socket_address: {address: 0.0.0.0, port_value: 15000}

dynamic_resources:
  ads_config:
    api_type: GRPC
    grpc_services:
    - envoy_grpc:
        cluster_name: ads
    set_node_on_first_message_only: true
  cds_config:
    ads: {}
  lds_config:
    ads: {}

static_resources:
  clusters:

  - name: ads
    connect_timeout: 0.25s
    type: LOGICAL_DNS
    http2_protocol_options: {}
    tls_context:
      common_tls_context:
        alpn_protocols:
          - h2
        validation_context:
          trusted_ca: { filename: "/etc/ssl/certs/root-cert.pem" }
        tls_params:
          tls_minimum_protocol_version: TLSv1_2
          tls_maximum_protocol_version: TLSv1_3
          cipher_suites: "[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]"
        tls_certificates:
          - certificate_chain: { filename: "/etc/ssl/certs/cert.pem" }
            private_key: { filename: "/etc/ssl/certs/key.pem" }
    load_assignment:
      cluster_name: ads
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: %s
                port_value: %d

  - name: metrics_server
    connect_timeout: 0.25s
    type: LOGICAL_DNS
    http2_protocol_options: {}
    tls_context:
      common_tls_context:
        alpn_protocols:
          - h2
        validation_context:
          trusted_ca: { filename: "/etc/ssl/certs/root-cert.pem" }
        tls_params:
          tls_minimum_protocol_version: TLSv1_2
          tls_maximum_protocol_version: TLSv1_3
          cipher_suites: "[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]"
        tls_certificates:
          - certificate_chain: { filename: "/etc/ssl/certs/cert.pem" }
            private_key: { filename: "/etc/ssl/certs/key.pem" }
    load_assignment:
      cluster_name: metrics_server
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: %s
                port_value: %d

stats_sinks:
  - name: envoy.metrics_service
    config:
      grpc_service:
        envoy_grpc:
          cluster_name: metrics_server

---`, xdsHost, common.AggregatedDiscoveryServicePort, xdsHost, common.MetricsServicePort)

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
		glog.Infof("Updating configMap for envoy bootstrap config: name=%s, namespace=%s", name, namespace)
		existing.Data = configMap.Data
		return wh.kubeClient.CoreV1().ConfigMaps(namespace).Update(existing)
	}

	glog.Infof("Creating configMap for envoy boostrap config: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().ConfigMaps(namespace).Create(configMap)
}

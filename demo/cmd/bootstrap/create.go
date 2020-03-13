package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/open-service-mesh/osm/demo/cmd/common"
)

const (
	configMapName = "envoyproxy-config"
)

func main() {
	namespace := os.Getenv(common.KubeNamespaceEnvVar)
	if namespace == "" {
		fmt.Println("Empty namespace")
		os.Exit(1)
	}
	clientset := getClient()

	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(configMapName, &metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		fmt.Println("Error deleting config map: ", err)
	}

	xdsHost := fmt.Sprintf("ads.%s.svc.cluster.local", namespace)
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
                port_value: 15128
---`, xdsHost)

	path := filepath.Join(".", "config")
	os.MkdirAll(path, os.ModePerm)
	fileName := fmt.Sprintf("%s/bootstrap.yaml", path)
	if err := ioutil.WriteFile(fileName, []byte(yamlContent), 0644); err != nil {
		fmt.Printf("Error writing to %s: %s", fileName, err)
	}

	configMap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"bootstrap.yaml": yamlContent,
		},
		BinaryData: nil,
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(configMap)
	if err != nil {
		fmt.Println("Error creating config map: ", err)
		os.Exit(1)
	}
}

func getClient() *kubernetes.Clientset {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv(common.KubeConfigEnvVar)
	if kubeConfigFile != "" {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
		if err != nil {
			fmt.Printf("Error fetching Kubernetes config. Ensure correctness of CLI argument 'kubeconfig=%s': %s", kubeConfigFile, err)
			os.Exit(1)
		}
	} else {
		// creates the in-cluster config
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("Error generating Kubernetes config: %s", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		fmt.Println("error in getting access to K8S")
		os.Exit(1)
	}
	return clientset
}

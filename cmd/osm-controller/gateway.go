package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/identity"
	"github.com/openservicemesh/osm/pkg/multicluster"
	"github.com/openservicemesh/osm/pkg/utils"
)

const (
	gatewayBootstrapSecretName = "osm-multicluster-gateway-bootstrap-config" // #nosec G101: Potential hardcoded credentials
	bootstrapConfigKey         = "bootstrap.yaml"
)

func bootstrapOSMMulticlusterGateway(kubeClient kubernetes.Interface, certManager certificate.Manager, osmNamespace string) error {
	secret, err := kubeClient.CoreV1().Secrets(osmNamespace).Get(context.Background(), gatewayBootstrapSecretName, metav1.GetOptions{})
	if err != nil {
		return errors.Errorf("Error fetching OSM gateway's bootstrap config %s/%s", osmNamespace, gatewayBootstrapSecretName)
	}

	if bootstrapData, ok := secret.Data[bootstrapConfigKey]; !ok {
		return errors.Errorf("Missing OSM gateway bootstrap config in %s/%s", osmNamespace, gatewayBootstrapSecretName)
	} else if isValidBootstrapData(bootstrapData) {
		// If there is a valid bootstrap config, it means we do not need to reconfigure it. It implies
		// osm-controller restarted after creating the bootstrap config previously.
		log.Info().Msg("A valid bootstrap config exists for OSM gateway, skipping gateway bootstrapping")
		return nil
	}

	gatewayCN := multicluster.GetMulticlusterGatewaySubjectCommonName(osmServiceAccount, osmNamespace)
	bootstrapCert, err := certManager.IssueCertificate(gatewayCN, constants.XDSCertificateValidityPeriod)
	if err != nil {
		return errors.Errorf("Error issuing bootstrap certificate for OSM gateway: %s", err)
	}

	bootstrapConfig, err := bootstrap.BuildFromConfig(bootstrap.Config{
		NodeID:           bootstrapCert.GetCommonName().String(),
		AdminPort:        constants.EnvoyAdminPort,
		XDSClusterName:   constants.OSMControllerName,
		XDSHost:          fmt.Sprintf("%s.%s.svc.%s", constants.OSMControllerName, osmNamespace, identity.ClusterLocalTrustDomain),
		XDSPort:          constants.ADSServerPort,
		TrustedCA:        bootstrapCert.GetIssuingCA(),
		CertificateChain: bootstrapCert.GetCertificateChain(),
		PrivateKey:       bootstrapCert.GetPrivateKey(),
	})
	if err != nil {
		return errors.Errorf("Error building OSM gateway's bootstrap config from %s/%s", osmNamespace, gatewayBootstrapSecretName)
	}

	bootstrapData, err := utils.ProtoToYAML(bootstrapConfig)
	if err != nil {
		return errors.Errorf("Error marshalling updated OSM gateway's bootstrap config from %s/%s", osmNamespace, gatewayBootstrapSecretName)
	}

	updatedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayBootstrapSecretName,
			Namespace: osmNamespace,
		},
		Data: map[string][]byte{
			bootstrapConfigKey: bootstrapData,
		},
	}

	patchJSON, err := json.Marshal(updatedSecret)
	if err != nil {
		return err
	}

	if _, err = kubeClient.CoreV1().Secrets(osmNamespace).Patch(context.Background(), gatewayBootstrapSecretName, types.StrategicMergePatchType, patchJSON, metav1.PatchOptions{}); err != nil {
		return errors.Errorf("Error patching OSM gateway's bootstrap secret %s/%s: %s", osmNamespace, gatewayBootstrapSecretName, err)
	}

	return nil
}

// isValidBootstrapData returns a boolean indicating if the bootstrap config YAML data is valid
// It determines validity by examining the presence of the minimum set of top level keys that
// should be present in the bootstrap config YAML. This is enough to determine validity because
// it means these top level keys were previously encoded as a part of the k8s secret corresponding
// to the bootstrap config.
func isValidBootstrapData(bootstrapData []byte) bool {
	js, err := yaml.YAMLToJSON([]byte(bootstrapData))
	if err != nil {
		return false
	}

	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(js, &jsonMap)
	if err != nil {
		return false
	}

	requiredKeys := []string{"admin", "node", "static_resources", "dynamic_resources"}
	for _, key := range requiredKeys {
		if _, ok := jsonMap[key]; !ok {
			return false
		}
	}

	return true
}

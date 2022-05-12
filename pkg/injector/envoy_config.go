package injector

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	xds_cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	xds_listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/utils"
	"github.com/openservicemesh/osm/pkg/version"
)

func generateEnvoyConfig(config envoyBootstrapConfigMeta, cfg configurator.Configurator) (*xds_bootstrap.Bootstrap, error) {
	bootstrapConfig, err := bootstrap.BuildFromConfig(bootstrap.Config{
		NodeID:                config.NodeID,
		AdminPort:             constants.EnvoyAdminPort,
		XDSClusterName:        constants.OSMControllerName,
		XDSHost:               config.XDSHost,
		XDSPort:               config.XDSPort,
		TLSMinProtocolVersion: config.TLSMinProtocolVersion,
		TLSMaxProtocolVersion: config.TLSMaxProtocolVersion,
		CipherSuites:          config.CipherSuites,
		ECDHCurves:            config.ECDHCurves,
	})
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy boostrap config")
		return nil, err
	}

	probeListeners, probeClusters, err := getProbeResources(config)
	if err != nil {
		return nil, err
	}
	bootstrapConfig.StaticResources.Listeners = append(bootstrapConfig.StaticResources.Listeners, probeListeners...)
	bootstrapConfig.StaticResources.Clusters = append(bootstrapConfig.StaticResources.Clusters, probeClusters...)

	return bootstrapConfig, nil
}

func getTLSSDSConfigYAML() ([]byte, error) {
	tlsSDSConfig, err := bootstrap.BuildTLSSecret()
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy TLS Certificate SDS Config")
		return nil, err
	}

	configYAML, err := utils.ProtoToYAML(tlsSDSConfig)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal Envoy TLS Certificate SDS Config to yaml")
		return nil, err
	}
	return configYAML, nil
}

func getValidationContextSDSConfigYAML() ([]byte, error) {
	validationContextSDSConfig, err := bootstrap.BuildValidationSecret()
	if err != nil {
		log.Error().Err(err).Msgf("Error building Envoy Validation Context SDS Config")
		return nil, err
	}

	configYAML, err := utils.ProtoToYAML(validationContextSDSConfig)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal Envoy Validation Context SDS Config to yaml")
		return nil, err
	}
	return configYAML, nil
}

// getProbeResources returns the listener and cluster objects that are statically configured to serve
// startup, readiness and liveness probes.
// These will not change during the lifetime of the Pod.
// If the original probe defined a TCPSocket action, listener and cluster objects are not configured
// to serve that probe.
func getProbeResources(config envoyBootstrapConfigMeta) ([]*xds_listener.Listener, []*xds_cluster.Cluster, error) {
	// This slice is the list of listeners for liveness, readiness, startup IF these have been configured in the Pod Spec
	var listeners []*xds_listener.Listener
	var clusters []*xds_cluster.Cluster

	// Is there a liveness probe in the Pod Spec?
	if config.OriginalHealthProbes.liveness != nil && !config.OriginalHealthProbes.liveness.isTCPSocket {
		listener, err := getLivenessListener(config.OriginalHealthProbes.liveness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting liveness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getLivenessCluster(config.OriginalHealthProbes.liveness))
	}

	// Is there a readiness probe in the Pod Spec?
	if config.OriginalHealthProbes.readiness != nil && !config.OriginalHealthProbes.readiness.isTCPSocket {
		listener, err := getReadinessListener(config.OriginalHealthProbes.readiness)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting readiness listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getReadinessCluster(config.OriginalHealthProbes.readiness))
	}

	// Is there a startup probe in the Pod Spec?
	if config.OriginalHealthProbes.startup != nil && !config.OriginalHealthProbes.startup.isTCPSocket {
		listener, err := getStartupListener(config.OriginalHealthProbes.startup)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting startup listener")
			return nil, nil, err
		}
		listeners = append(listeners, listener)
		clusters = append(clusters, getStartupCluster(config.OriginalHealthProbes.startup))
	}

	return listeners, clusters, nil
}

// This will read an existing envoy bootstrap config, and create a new copy by changing the NodeID, and certificates.
func (wh *mutatingWebhook) createEnvoyBootstrapFromExisting(newBootstrapSecretName, oldBootstrapSecretName, namespace string, cert *certificate.Certificate) (*corev1.Secret, error) {
	existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), oldBootstrapSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	yamlBytes := existing.Data[bootstrap.EnvoyBootstrapConfigFile]

	config := &xds_bootstrap.Bootstrap{}
	if err := utils.YAMLToProto(yamlBytes, config); err != nil {
		return nil, fmt.Errorf("error unmarshalling envoy bootstrap config: %w", err)
	}

	config.Node.Id = cert.GetCommonName().String()

	return wh.marshalAndSaveBootstrap(newBootstrapSecretName, namespace, config, cert)
}

func (wh *mutatingWebhook) createEnvoyBootstrapConfig(name, namespace, osmNamespace string, cert *certificate.Certificate, originalHealthProbes healthProbes) (*corev1.Secret, error) {
	configMeta := envoyBootstrapConfigMeta{
		EnvoyAdminPort: constants.EnvoyAdminPort,
		XDSClusterName: constants.OSMControllerName,
		NodeID:         cert.GetCommonName().String(),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.OSMControllerName, osmNamespace),
		XDSPort: constants.ADSServerPort,

		// OriginalHealthProbes stores the path and port for liveness, readiness, and startup health probes as initially
		// defined on the Pod Spec.
		OriginalHealthProbes: originalHealthProbes,

		TLSMinProtocolVersion: wh.configurator.GetMeshConfig().Spec.Sidecar.TLSMinProtocolVersion,
		TLSMaxProtocolVersion: wh.configurator.GetMeshConfig().Spec.Sidecar.TLSMaxProtocolVersion,
		CipherSuites:          wh.configurator.GetMeshConfig().Spec.Sidecar.CipherSuites,
		ECDHCurves:            wh.configurator.GetMeshConfig().Spec.Sidecar.ECDHCurves,
	}
	config, err := generateEnvoyConfig(configMeta, wh.configurator)
	if err != nil {
		return nil, err
	}
	// marshalAndSaveBootstrap
	return wh.marshalAndSaveBootstrap(name, namespace, config, cert)
}

func (wh *mutatingWebhook) marshalAndSaveBootstrap(name, namespace string, config *xds_bootstrap.Bootstrap, cert *certificate.Certificate) (*corev1.Secret, error) {
	configYAML, err := utils.ProtoToYAML(config)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal envoy bootstrap config to yaml")
		return nil, err
	}

	tlsYamlContent, err := getTLSSDSConfigYAML()
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy TLS Certificate SDS Config YAML")
		return nil, err
	}

	validationYamlContent, err := getValidationContextSDSConfigYAML()
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy Validation Context SDS Config YAML")
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:     constants.OSMAppNameLabelValue,
				constants.OSMAppInstanceLabelKey: wh.meshName,
				constants.OSMAppVersionLabelKey:  version.Version,
			},
		},
		Data: map[string][]byte{
			bootstrap.EnvoyBootstrapConfigFile:            configYAML,
			bootstrap.EnvoyTLSCertificateSDSSecretFile:    tlsYamlContent,
			bootstrap.EnvoyValidationContextSDSSecretFile: validationYamlContent,
			bootstrap.EnvoyXDSCACertFile:                  cert.GetIssuingCA(),
			bootstrap.EnvoyXDSCertFile:                    cert.GetCertificateChain(),
			bootstrap.EnvoyXDSKeyFile:                     cert.GetPrivateKey(),
		},
	}

	log.Debug().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}

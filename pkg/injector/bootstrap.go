package injector

import (
	"context"
	"fmt"
	"reflect"
	"time"

	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/envoy/registry"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/utils"
	"github.com/openservicemesh/osm/pkg/version"
)

// This will read an existing envoy bootstrap config, and create a new copy by changing the NodeID, and certificates.
func (wh *mutatingWebhook) createEnvoyBootstrapFromExisting(proxyUUID uuid.UUID, oldBootstrapSecretName, namespace string, cert *certificate.Certificate) (*corev1.Secret, error) {
	existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(context.Background(), oldBootstrapSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	yamlBytes := existing.Data[bootstrap.EnvoyBootstrapConfigFile]

	config := &xds_bootstrap.Bootstrap{}
	if err := utils.YAMLToProto(yamlBytes, config); err != nil {
		return nil, fmt.Errorf("error unmarshalling envoy bootstrap config: %w", err)
	}

	config.Node.Id = proxyUUID.String()

	return wh.marshalAndSaveBootstrap(bootstrapConfigName(proxyUUID), namespace, config, cert)
}

func (wh *mutatingWebhook) createEnvoyBootstrapConfig(proxyUUID uuid.UUID, namespace string, cert *certificate.Certificate, originalHealthProbes map[string]models.HealthProbes) (*corev1.Secret, error) {
	builder := bootstrap.Builder{
		NodeID: proxyUUID.String(),

		XDSHost: fmt.Sprintf("%s.%s.svc.cluster.local", constants.OSMControllerName, wh.osmNamespace),

		// OriginalHealthProbes stores the path and port for liveness, readiness, and startup health probes as initially
		// defined on the Pod Spec.
		OriginalHealthProbes: originalHealthProbes,

		TLSMinProtocolVersion: wh.kubeController.GetMeshConfig().Spec.Sidecar.TLSMinProtocolVersion,
		TLSMaxProtocolVersion: wh.kubeController.GetMeshConfig().Spec.Sidecar.TLSMaxProtocolVersion,
		CipherSuites:          wh.kubeController.GetMeshConfig().Spec.Sidecar.CipherSuites,
		ECDHCurves:            wh.kubeController.GetMeshConfig().Spec.Sidecar.ECDHCurves,
	}
	bootstrapConfig, err := builder.Build()
	if err != nil {
		return nil, err
	}

	return wh.marshalAndSaveBootstrap(bootstrapConfigName(proxyUUID), namespace, bootstrapConfig, cert)
}

func (wh *mutatingWebhook) marshalAndSaveBootstrap(name, namespace string, config *xds_bootstrap.Bootstrap, cert *certificate.Certificate) (*corev1.Secret, error) {
	configYAML, err := utils.ProtoToYAML(config)
	if err != nil {
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrMarshallingProtoToYAML)).
			Msgf("Failed to marshal envoy bootstrap config to yaml")
		return nil, err
	}

	tlsYamlContent, err := bootstrap.GetTLSSDSConfigYAML()
	if err != nil {
		log.Error().Err(err).Msg("Error creating Envoy TLS Certificate SDS Config YAML")
		return nil, err
	}

	validationYamlContent, err := bootstrap.GetValidationContextSDSConfigYAML()
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
			bootstrap.EnvoyXDSCACertFile:                  cert.GetTrustedCAs(),
			bootstrap.EnvoyXDSCertFile:                    cert.GetCertificateChain(),
			bootstrap.EnvoyXDSKeyFile:                     cert.GetPrivateKey(),
		},
	}

	log.Debug().Msgf("Creating bootstrap config for Envoy: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
}

// NewBootstrapSecretRotator returns a new bootstrap secret rotator.
func NewBootstrapSecretRotator(kubeController k8s.Controller, proxyRegistry *registry.ProxyRegistry, certManager *certificate.Manager, checkInterval time.Duration) *BootstrapSecretRotator {
	return &BootstrapSecretRotator{
		kubeController: kubeController,
		proxyRegistry:  proxyRegistry,
		certManager:    certManager,
		checkInterval:  checkInterval,
	}
}

// rotateBootstrapSecrets updates the bootstrap secret from the connectedProxy by
// getting the current or issuing a new certificate.
func (b *BootstrapSecretRotator) rotateBootstrapSecrets(ctx context.Context) {
	proxies := b.proxyRegistry.ListConnectedProxies()
	for _, proxy := range proxies {
		envoyBootstrapConfigName := bootstrapSecretPrefix + proxy.UUID.String()
		secret, err := b.kubeController.GetSecret(ctx, envoyBootstrapConfigName, proxy.PodMetadata.Namespace)
		if err != nil {
			log.Error().Err(err).Msgf("Error getting the secret %s/%s", proxy.PodMetadata.Namespace, envoyBootstrapConfigName)
			continue
		}

		cnPrefix := envoy.NewXDSCertCNPrefix(proxy.UUID, envoy.KindSidecar, proxy.Identity)
		bootstrapCert, err := b.certManager.IssueCertificate(cnPrefix, certificate.Internal)
		if err != nil {
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
				Msgf("Error rotating cert %s", bootstrapCert)
		}

		ca := bootstrapCert.GetTrustedCAs()
		certChain := bootstrapCert.GetCertificateChain()
		pk := bootstrapCert.GetPrivateKey()

		// if the secret and issued cert are the same no need to update the secret
		if reflect.DeepEqual(ca, secret.Data["ca.crt"]) && reflect.DeepEqual(pk, secret.Data["tls.key"]) && reflect.DeepEqual(certChain, secret.Data["tls.crt"]) {
			continue
		}
		secretData := map[string][]byte{
			"ca.crt":  ca,
			"tls.crt": certChain,
			"tls.key": pk,
		}
		err = b.kubeController.UpdateSecret(ctx, secret, secretData)
		if err != nil {
			if apierrors.IsConflict(err) {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingBootstrapSecret)).
					Msgf("There was an update conflict while trying to update the envoy bootstrap config secret %s/%s with issued cert %s", secret.Namespace, secret.Name, bootstrapCert)
				continue
			}
			log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingBootstrapSecret)).
				Msgf("Error updating bootstrap secret %s/%s with issued cert %s", secret.Namespace, secret.Name, bootstrapCert)
			continue
		}
	}
}

// StartBootstrapSecretRotationTicker will start a ticker to check if the bootstrap secrets should be
// updated every BootstrapSecretRotator check interval
func (b *BootstrapSecretRotator) StartBootstrapSecretRotationTicker(ctx context.Context) {
	ticker := time.NewTicker(b.checkInterval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				b.rotateBootstrapSecrets(ctx)
			}
		}
	}()
}

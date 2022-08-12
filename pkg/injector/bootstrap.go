package injector

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	xds_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/envoy/bootstrap"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
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
func NewBootstrapSecretRotator(ctx context.Context, kubeClient kubernetes.Interface, informerCollection *informers.InformerCollection, certManager *certificate.Manager, checkInterval time.Duration) *BootstrapSecretRotator {
	return &BootstrapSecretRotator{
		context:            ctx,
		kubeClient:         kubeClient,
		informerCollection: informerCollection,
		certManager:        certManager,
		checkInterval:      checkInterval,
	}
}

// listBootstrapSecrets returns the bootstrap secrets stored in the informerCollection's store.
// this function is used to get the namespace of the secrets.
func (b *BootstrapSecretRotator) listBootstrapSecrets() []*corev1.Secret {
	// informers return slice of pointers so we'll convert them to value types before returning
	secretPtrs := b.informerCollection.List(informers.InformerKeySecret)
	var secrets []*corev1.Secret

	for _, secretPtr := range secretPtrs {
		if secretPtr == nil {
			continue
		}
		secret, ok := secretPtr.(*corev1.Secret)
		if !ok {
			continue
		}
		// finds bootstrap secrets
		if strings.Contains(secret.Name, bootstrapSecretPrefix) {
			secrets = append(secrets, secret)
		}
	}

	return secrets
}

// rotateBootstrapSecrets finds the bootstrap secret of the given certificate
// from the list of secrets stored in the informerCollection's store and updates the secret.
func (b *BootstrapSecretRotator) rotateBootstrapSecrets() {
	secrets := b.listBootstrapSecrets()
	for _, secret := range secrets {
		secretProxyUUID := strings.ReplaceAll(secret.Name, bootstrapSecretPrefix, "")
		certs := b.certManager.ListIssuedCertificates()
		for _, cert := range certs {
			// CommonName for a xds certificate has the form: <ProxyUUID>.<kind>.<identity>
			certProxyUUID := strings.Split(cert.CommonName.String(), ".")[0]
			// checks if this is the corresponding cert for the secret
			if secretProxyUUID != certProxyUUID {
				continue
			}
			opts := []certificate.IssueOption{certificate.FullCNProvided()}
			c, err := b.certManager.IssueCertificate(cert.CommonName.String(), cert.CertType, opts...)
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrRotatingCert)).
					Msgf("Error rotating cert SerialNumber=%s", cert.GetSerialNumber())
			}
			// if the secret and issued cert are the same no need to update the secret
			if reflect.DeepEqual(c.GetIssuingCA, secret.Data["ca.crt"]) && reflect.DeepEqual(c.PrivateKey, secret.Data["tls.key"]) && reflect.DeepEqual(c.CertChain, secret.Data["tls.crt"]) {
				continue
			}
			// TODO: check mapping
			secretData := map[string][]byte{
				"ca.crt":  cert.GetIssuingCA(),
				"tls.crt": cert.GetCertificateChain(),
				"tls.key": cert.GetPrivateKey(),
			}
			newSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
				Type: corev1.SecretTypeTLS,
				Data: secretData,
			}
			_, err = b.kubeClient.CoreV1().Secrets(secret.GetNamespace()).Update(b.context, newSecret, metav1.UpdateOptions{})
			if err != nil {
				log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrUpdatingBootstrapSecret)).
					Msgf("Error updating bootstrap secret %s/%s with issued cert %s", secret.Namespace, secret.Name, cert.CommonName.String())
				continue
			}
		}
	}
}

// StartBootstrapSecretRotationTicker will start a ticker to check if the bootstrap secrets should be
// updated every BootstrapSecretRotator check interval
func (b *BootstrapSecretRotator) StartBootstrapSecretRotationTicker() {
	ticker := time.NewTicker(b.checkInterval)
	go func() {
		for {
			select {
			case <-b.context.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				b.rotateBootstrapSecrets()
			}
		}
	}()
}

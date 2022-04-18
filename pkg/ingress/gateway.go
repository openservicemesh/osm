package ingress

import (
	"context"
	"reflect"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/announcements"
	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"
	"github.com/openservicemesh/osm/pkg/k8s/events"

	"github.com/openservicemesh/osm/pkg/certificate"
)

// provisionIngressGatewayCert does the following:
// 1. If an ingress gateway certificate spec is specified in the MeshConfig resource, issues a certificate
//    for it and stores it in the referenced secret.
// 2. Starts a goroutine to watch for changes to the MeshConfig resource and certificate rotation, and
//    updates/removes the certificate and secret as necessary.
func (c *client) provisionIngressGatewayCert(stop <-chan struct{}) error {
	defaultCertSpec := c.cfg.GetMeshConfig().Spec.Certificate.IngressGateway
	if defaultCertSpec != nil {
		// Issue a certificate for the default certificate spec
		if err := c.createAndStoreGatewayCert(*defaultCertSpec); err != nil {
			return errors.Wrap(err, "Error provisioning default ingress gateway cert")
		}
	}

	// Initialize a watcher to watch for CREATE/UPDATE/DELETE on the ingress gateway cert spec
	go c.handleCertificateChange(defaultCertSpec, stop)

	return nil
}

// createAndStoreGatewayCert creates a certificate for the given certificate spec and stores
// it in the referenced k8s secret if the spec is valid.
func (c *client) createAndStoreGatewayCert(spec configv1alpha3.IngressGatewayCertSpec) error {
	if len(spec.SubjectAltNames) == 0 {
		return errors.New("Ingress gateway certificate spec must specify at least 1 SAN")
	}

	// Validate the validity duration
	certValidityDuration, err := time.ParseDuration(spec.ValidityDuration)
	if err != nil {
		return errors.Wrapf(err, "Invalid cert duration '%s' specified", spec.ValidityDuration)
	}

	// Validate the secret ref
	if spec.Secret.Name == "" || spec.Secret.Namespace == "" {
		return errors.Errorf("Ingress gateway cert secret's name and namespace cannot be nil, got %s/%s", spec.Secret.Namespace, spec.Secret.Name)
	}

	// Issue a certificate
	// OSM only support configuring a single SAN per cert, so pick the first one
	certCN := certificate.CommonName(spec.SubjectAltNames[0])

	// A certificate for this CN may be cached already. Delete it before issuing a new certificate.
	c.certProvider.ReleaseCertificate(certCN)
	issuedCert, err := c.certProvider.IssueCertificate(certCN, certValidityDuration)
	if err != nil {
		return errors.Wrapf(err, "Error issuing a certificate for ingress gateway")
	}

	// Store the certificate in the referenced secret
	if err := c.storeCertInSecret(issuedCert, spec.Secret); err != nil {
		return errors.Wrapf(err, "Error storing ingress gateway cert in secret %s/%s", spec.Secret.Namespace, spec.Secret.Name)
	}

	return nil
}

// storeCertInSecret stores the certificate in the specified k8s TLS secret
func (c *client) storeCertInSecret(cert *certificate.Certificate, secret corev1.SecretReference) error {
	secretData := map[string][]byte{
		"ca.crt":  cert.GetIssuingCA(),
		"tls.crt": cert.GetCertificateChain(),
		"tls.key": cert.GetPrivateKey(),
	}

	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: secret.Namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: secretData,
	}

	_, err := c.kubeClient.CoreV1().Secrets(secret.Namespace).Create(context.Background(), sec, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		_, err = c.kubeClient.CoreV1().Secrets(secret.Namespace).Update(context.Background(), sec, metav1.UpdateOptions{})
	}
	return err
}

// handleCertificateChange updates the gateway certificate and secret when the MeshConfig resource changes or
// when the corresponding gateway certificate is rotated.
func (c *client) handleCertificateChange(currentCertSpec *configv1alpha3.IngressGatewayCertSpec, stop <-chan struct{}) {
	kubePubSub := c.msgBroker.GetKubeEventPubSub()
	meshConfigUpdateChan := kubePubSub.Sub(announcements.MeshConfigUpdated.String())
	defer c.msgBroker.Unsub(kubePubSub, meshConfigUpdateChan)

	certPubSub := c.msgBroker.GetCertPubSub()
	certRotateChan := certPubSub.Sub(announcements.CertificateRotated.String())
	defer c.msgBroker.Unsub(certPubSub, certRotateChan)

	for {
		select {
		// MeshConfig was updated
		case msg, ok := <-meshConfigUpdateChan:
			if !ok {
				log.Warn().Msgf("Notification channel closed for MeshConfig")
				continue
			}

			event, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
				continue
			}

			updatedMeshConfig, ok := event.NewObj.(*configv1alpha3.MeshConfig)
			if !ok {
				log.Error().Msgf("Received unexpected object %T, expected MeshConfig", updatedMeshConfig)
				continue
			}
			newCertSpec := updatedMeshConfig.Spec.Certificate.IngressGateway
			if reflect.DeepEqual(currentCertSpec, newCertSpec) {
				log.Debug().Msg("Ingress gateway certificate spec was not updated")
				continue
			}
			if newCertSpec == nil && currentCertSpec != nil {
				// Implies the certificate reference was removed, delete the corresponding secret and certificate
				if err := c.removeGatewayCertAndSecret(*currentCertSpec); err != nil {
					log.Error().Err(err).Msg("Error removing stale gateway certificate/secret")
				}
			} else if newCertSpec != nil {
				// New cert spec is not nil and is not the same as the current cert spec, update required
				err := c.createAndStoreGatewayCert(*newCertSpec)
				if err != nil {
					log.Error().Err(err).Msgf("Error updating ingress gateway cert and secret")
				}
			}
			currentCertSpec = newCertSpec

		// A certificate was rotated
		case msg, ok := <-certRotateChan:
			if !ok {
				log.Warn().Msg("Notification channel closed for certificate rotation")
				continue
			}

			event, ok := msg.(events.PubSubMessage)
			if !ok {
				log.Error().Msgf("Received unexpected message %T on channel, expected PubSubMessage", event)
				continue
			}
			cert, ok := event.NewObj.(*certificate.Certificate)
			if !ok {
				log.Error().Msgf("Received unexpected message %T on cert rotation channel, expected Certificate", cert)
				continue
			}

			// This should never happen, but guarding against a panic due to the usage of a pointer
			if currentCertSpec == nil {
				log.Error().Msgf("Current ingress gateway cert spec is nil, but a certificate for it was rotated - unexpected")
				continue
			}

			cnInCertSpec := currentCertSpec.SubjectAltNames[0] // Only single SAN is supported in certs

			// Only update the secret if the cert rotated matches the cert spec
			if cert.GetCommonName() != certificate.CommonName(cnInCertSpec) {
				continue
			}

			log.Info().Msg("Ingress gateway certificate was rotated, updating corresponding secret")
			if err := c.createAndStoreGatewayCert(*currentCertSpec); err != nil {
				log.Error().Err(err).Msgf("Error updating ingress gateway cert secret after cert rotation")
			}

		case <-stop:
			return
		}
	}
}

// removeGatewayCertAndSecret removes the secret and certificate corresponding to the existing cert spec
func (c *client) removeGatewayCertAndSecret(storedCertSpec configv1alpha3.IngressGatewayCertSpec) error {
	err := c.kubeClient.CoreV1().Secrets(storedCertSpec.Secret.Namespace).Delete(context.Background(), storedCertSpec.Secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	certCN := certificate.CommonName(storedCertSpec.SubjectAltNames[0]) // Only single SAN is supported in certs
	c.certProvider.ReleaseCertificate(certCN)

	return nil
}

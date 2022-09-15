package ingress

import (
	"context"
	"reflect"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/k8s/events"

	"github.com/openservicemesh/osm/pkg/certificate"
)

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

func (c *client) provisionIngressGatewayCert(currentCertSpec *configv1alpha2.IngressGatewayCertSpec, stop <-chan struct{}) {
	meshConfigUpdateChan, unsub := c.msgBroker.SubscribeKubeEvents(events.MeshConfig.Updated())
	defer unsub()

	// stopWatchingRotations is a function that should be called when the cert rotation is no longer needed on the
	// specified SAN. This can happen on SAN changes or on the deletion of the ingress gateway spec.
	// It's set to an empty func so that when a SAN is not specified we can safely call stop without checking for a nil
	// value.
	stopWatchingRotations := func() {}

	if currentCertSpec != nil && len(currentCertSpec.SubjectAltNames) > 0 {
		stopWatchingRotations = c.handleCertChanges(currentCertSpec.SubjectAltNames[0], currentCertSpec.Secret)
	}

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

			updatedMeshConfig, ok := event.NewObj.(*configv1alpha2.MeshConfig)
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
				stopWatchingRotations()
				// Implies the certificate reference was removed, delete the corresponding secret and certificate
				if err := c.removeGatewayCertAndSecret(*currentCertSpec); err != nil {
					log.Error().Err(err).Msg("Error removing stale gateway certificate/secret")
				}
			} else if shouldRelisten(currentCertSpec, newCertSpec) {
				stopWatchingRotations()
				stopWatchingRotations = c.handleCertChanges(newCertSpec.SubjectAltNames[0], newCertSpec.Secret)
			}
			currentCertSpec = newCertSpec

		case <-stop:
			stopWatchingRotations()
			return
		}
	}
}

func shouldRelisten(oldSpec *configv1alpha2.IngressGatewayCertSpec, newSpec *configv1alpha2.IngressGatewayCertSpec) bool {
	if newSpec == nil || len(newSpec.SubjectAltNames) == 0 {
		return false
	}

	if oldSpec == nil || len(oldSpec.SubjectAltNames) == 0 {
		return true
	}

	return oldSpec.SubjectAltNames[0] != newSpec.SubjectAltNames[0]
}

// handleCertChanges creates and stores a certificate with the given common name into the given secret ref.
// It then watches for all cert rotations and updates the secret on changes.
func (c *client) handleCertChanges(cn string, secret corev1.SecretReference) func() {
	// This is some fanciness to provide the following guarantees:
	// 1. The subscription is called before a goroutine starts, allowing the subscription creation to be deterministic.
	//	  This means that the subscription is guaranteed to not miss any rotations in the period between issuance and
	//    the goroutine starting.
	// 2. By signalling to stop (instead of just closing it), we guarantee the close func doesn't return before the
	// 	  goroutine stops. This prevents race conditions where we return before the below goroutine has exited, spawn
	//	  a new routine due to a changed SAN, and both routines can now race to write out the certificate. This can occur
	//	  when an update with a new SAN comes through at the same time as a rotation is triggered.
	// 3. The once allows the close method to be called multiple times.
	// #1 and #2 are required to prevent
	var once sync.Once
	stop := make(chan struct{})

	// must subscribe prior to issuing the cert to guarantee we get all rotations.
	certRotateChan, unsub := c.certProvider.SubscribeRotations(cn)

	cert, err := c.certProvider.IssueCertificate(certificate.ForIngressGateway(cn))
	if err != nil {
		log.Err(err).Msg("error issuing a certificate for ingress gateway")
	}
	if err := c.storeCertInSecret(cert, secret); err != nil {
		log.Err(err).Msg("Error updating ingress gateway cert secret after cert rotation")
	}

	go func() {
		for {
			select {
			// A certificate was rotated
			case msg := <-certRotateChan:
				cert := msg.(*certificate.Certificate)
				log.Info().Msgf("Ingress gateway certificate was rotated, updating secret %s/%s", secret.Name, secret.Namespace)
				if err := c.storeCertInSecret(cert, secret); err != nil {
					log.Err(err).Msg("Error updating ingress gateway cert secret after cert rotation")
				}

			case <-stop:
				return
			}
		}
	}()
	return func() {
		// Allow close to be called multiple times by wrapping it in a once.
		once.Do(func() {
			// We don't just close the channel because we want to wait for the above goroutine to exit. Since there
			// is no logic (or race conditions), in the select's stop case we can safely proceed. See the method
			// comment for more.
			stop <- struct{}{}
			unsub()
			close(stop)
		})
	}
}

// removeGatewayCertAndSecret removes the secret and certificate corresponding to the existing cert spec
func (c *client) removeGatewayCertAndSecret(storedCertSpec configv1alpha2.IngressGatewayCertSpec) error {
	err := c.kubeClient.CoreV1().Secrets(storedCertSpec.Secret.Namespace).Delete(context.Background(), storedCertSpec.Secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	c.certProvider.ReleaseCertificate(storedCertSpec.SubjectAltNames[0]) // Only single SAN is supported in certs

	return nil
}

package k8s

import (
	"context"

	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/version"
)

// GetCertFromKubernetes is a helper function that loads a certificate from a Kubernetes secret
func GetCertFromKubernetes(ns string, secretName string, kubeClient kubernetes.Interface) (*certificate.Certificate, error) {
	certSecret, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingCertSecret)).
			Msgf("Could not retrieve certificate secret %q from namespace %q", secretName, ns)
		return nil, certificate.ErrSecretNotFound
	}

	pemCert, ok := certSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(certificate.ErrInvalidCertSecret).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrObtainingCertFromSecret)).
			Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretCAKey)
		return nil, certificate.ErrInvalidCertSecret
	}

	pemKey, ok := certSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]
	if !ok {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(certificate.ErrInvalidCertSecret).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrObtainingPrivateKeyFromSecret)).
			Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretRootPrivateKeyKey)
		return nil, certificate.ErrInvalidCertSecret
	}

	cert, err := certificate.NewFromPEM(pemCert, pemKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create new Certificate from PEM")
		return nil, err
	}

	return cert, nil
}

// GetCertificateFromSecret is a helper function that ensures creation and synchronization of a certificate
// using Kubernetes Secrets backend and API atomicity.
func GetCertificateFromSecret(ns string, secretName string, cert *certificate.Certificate, kubeClient kubernetes.Interface) (*certificate.Certificate, error) {
	// Attempt to create it in Kubernetes. When multiple agents attempt to create, only one of them will succeed.
	// All others will get "AlreadyExists" error back.
	secretData := map[string][]byte{
		constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
		constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:    constants.OSMAppNameLabelValue,
				constants.OSMAppVersionLabelKey: version.Version,
			},
		},
		Data: secretData,
	}

	if _, err := kubeClient.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{}); err == nil {
		log.Info().Msgf("Secret %s/%s created in kubernetes", ns, secretName)
	} else if apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("Secret %s/%s already exists in kubernetes, loading.", ns, secretName)
	} else {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCertSecret)).
			Msgf("Error creating/retrieving certificate secret %s/%s", ns, secretName)
		return nil, err
	}

	// For simplicity, we will load the certificate for all of them, this way the instance which created it
	// and the ones that didn't share the same code.
	cert, err := GetCertFromKubernetes(ns, secretName, kubeClient)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch certificate from Kubernetes")
		return nil, err
	}

	return cert, nil
}

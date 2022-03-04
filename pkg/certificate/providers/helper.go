package providers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/errcode"
	"github.com/openservicemesh/osm/pkg/version"
)

const (
	// Additional values for the root certificate
	rootCertCountry      = "US"
	rootCertLocality     = "CA"
	rootCertOrganization = "Open Service Mesh"
)

// TODO: pull this out into main, and dependency inject it into the certificate.Provider
func GetOrCreateTresorCA(namespace, secretName string, kubeClient kubernetes.Interface) (*certificate.Certificate, error) {
	rootCert, err := GetCertFromKubernetes(namespace, secretName, kubeClient)
	if err == nil {
		return rootCert, nil
	}
	log.Error().Err(err).Msg("Failed to get default CA, attempting to generate")

	// Now generate a Tresor cert.
	rootCert, err = tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)
	if err := CreateKubernetesCert(namespace, secretName, rootCert, kubeClient); apierrors.IsAlreadyExists(err) {
		log.Info().Msgf("Secret %s/%s already exists in kubernetes, loading.", namespace, secretName)
	} else if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCertSecret)).
			Msgf("Error creating/retrieving certificate secret %s/%s", namespace, secretName)
	}

	return GetCertFromKubernetes(namespace, secretName, kubeClient)
}

// TODO(steeling): put all k8s methods onto a single k8s client.
func CreateKubernetesCert(ns string, secretName string, cert *certificate.Certificate, kubeClient kubernetes.Interface) error {
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

	if _, err := kubeClient.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCertSecret)).
			Msgf("Error creating/retrieving certificate secret %s/%s", ns, secretName)
		return err
	}
	log.Info().Msgf("Secret %s/%s created in kubernetes", ns, secretName)
	return nil
}

// GetCertFromKubernetes is a helper function that loads a certificate from a Kubernetes secret
// The function returns an error only if a secret is found with invalid data.
func GetCertFromKubernetes(ns string, secretName string, kubeClient kubernetes.Interface) (*certificate.Certificate, error) {
	certSecret, err := kubeClient.CoreV1().Secrets(ns).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrFetchingCertSecret)).
			Msgf("Could not retrieve certificate secret %q from namespace %q", secretName, ns)
		return nil, errSecretNotFound
	}

	pemCert, ok := certSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(errInvalidCertSecret).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrObtainingCertFromSecret)).
			Msgf("Opaque k8s secret %s/%s does not have required field %q", ns, secretName, constants.KubernetesOpaqueSecretCAKey)
		return nil, errInvalidCertSecret
	}

	pemKey := certSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey]

	cert, err := certificate.NewFromPEM(pemCert, pemKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create new Certificate from PEM")
		return nil, err
	}

	return cert, nil
}

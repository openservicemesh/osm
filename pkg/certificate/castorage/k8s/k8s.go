package k8s

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/version"
)

// SecretClient is a client for reading and writing certificates to a single namespaced secret.
type SecretClient struct {
	kubeClient kubernetes.Interface

	version    string
	secretName string
	namespace  string
}

// NewCASecretClient returns a configured SecretClient.
func NewCASecretClient(kubeClient kubernetes.Interface, secretName, namespace, version string) *SecretClient {
	return &SecretClient{
		kubeClient: kubeClient,
		version:    version,
		secretName: secretName,
		namespace:  namespace,
	}
}

// Set will write the provided cert to the configured namespaced secret.
func (c *SecretClient) Set(ctx context.Context, cert *certificate.Certificate) (string, error) {
	// Attempt to create it in Kubernetes. When multiple agents attempt to create, only one of them will succeed.
	// All others will get "AlreadyExists" error back.
	secretData := map[string][]byte{
		constants.KubernetesOpaqueSecretCAKey: cert.GetCertificateChain(),
	}

	if cert.GetPrivateKey() != nil {
		secretData[constants.KubernetesOpaqueSecretRootPrivateKeyKey] = cert.GetPrivateKey()
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.secretName,
			Namespace: c.namespace,
			Labels: map[string]string{
				constants.OSMAppNameLabelKey:    constants.OSMAppNameLabelValue,
				constants.OSMAppVersionLabelKey: version.Version,
			},
		},
		Data: secretData,
	}

	_, err := c.kubeClient.CoreV1().Secrets(c.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return "", fmt.Errorf("k8s secret %s/%s already exists: %w", c.namespace, c.secretName, certificate.ErrSecretAlreadyExists)
	} else if err != nil {
		return "", fmt.Errorf("error setting k8s secret %s/%s: %w", c.namespace, c.secretName, err)
	}
	return c.version, nil
}

// Get will retrieve and attempt to convert the configured namespace secret to a certificate.Certificate.
func (c *SecretClient) Get(ctx context.Context) (*certificate.Certificate, error) {
	certSecret, err := c.kubeClient.CoreV1().Secrets(c.namespace).Get(ctx, c.secretName, metav1.GetOptions{})
	if err != nil {
		return nil, certificate.ErrInvalidCertSecret
	}

	pemCert, ok := certSecret.Data[constants.KubernetesOpaqueSecretCAKey]
	if !ok {
		return nil, certificate.ErrInvalidCertSecret
	}

	cert, err := certificate.NewFromPEM(pemCert, certSecret.Data[constants.KubernetesOpaqueSecretRootPrivateKeyKey])
	if err != nil {
		return nil, err
	}

	return cert, nil
}

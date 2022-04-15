package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/google/uuid"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests"
)

// go's time's aren't well-suited for equality checks, due to the timezones being different.
func assertCertsEqual(t *testing.T, expected *certificate.Certificate, actual *certificate.Certificate) {
	assert := tassert.New(t)

	assert.Equal(expected.CertChain, actual.CertChain)
	assert.Equal(expected.PrivateKey, actual.PrivateKey)
	assert.Equal(expected.SerialNumber, actual.SerialNumber)
	assert.Equal(expected.CommonName, actual.CommonName)
	assert.Equal(expected.IssuingCA, actual.IssuingCA)
	assert.WithinDuration(expected.Expiration, actual.Expiration, 0)
}

func TestSet(t *testing.T) {
	assert := tassert.New(t)
	kubeClient := fake.NewSimpleClientset()
	ctx := context.Background()

	// Create some cert, using tresor's api for simplicity
	cert, err := tresor.NewCA("common-name", time.Hour, "test-country", "test-locality", "test-org")
	assert.NoError(err)

	client := &SecretClient{
		kubeClient: kubeClient,
		namespace:  "test",
		secretName: "test",
		version:    "v1",
	}

	version, err := client.Set(ctx, cert)
	assert.NoError(err)
	assert.Equal(version, "v1")
	resCert, err := client.Get(ctx)
	assert.NoError(err)
	assertCertsEqual(t, cert, resCert)

	_, err = client.Set(ctx, cert)
	assert.True(errors.Is(err, certificate.ErrSecretAlreadyExists))
}

func TestGet(t *testing.T) {
	assert := tassert.New(t)

	certPEM, err := tests.GetPEMCert()
	assert.NoError(err)
	keyPEM, err := tests.GetPEMPrivateKey()
	assert.NoError(err)

	ns := uuid.New().String()
	secretName := uuid.New().String()

	client := &SecretClient{
		namespace:  ns,
		secretName: secretName,
		version:    "v3",
	}

	testCases := []struct {
		secret       *corev1.Secret
		expectError  bool
		expectNilVal bool
	}{
		{
			// Valid cert, valid test
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             certPEM,
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			},
			expectError:  false,
			expectNilVal: false,
		},
		{
			// Error when cert fetch is not present
			secret:       nil,
			expectError:  true,
			expectNilVal: true,
		},
		{
			// Error when CA key is missing
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			},
			expectError:  true,
			expectNilVal: true,
		},
		{
			// No error when Private Key is missing
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey: certPEM,
				},
			},
			expectError:  false,
			expectNilVal: false,
		},
	}
	ctx := context.Background()
	for _, testElement := range testCases {
		kubeClient := fake.NewSimpleClientset()
		client.kubeClient = kubeClient

		if testElement.secret != nil {
			_, err = kubeClient.CoreV1().Secrets(ns).Create(context.Background(), testElement.secret, metav1.CreateOptions{})
			assert.NoError(err)
		}

		cert, err := client.Get(ctx)

		assert.Equal(testElement.expectError, err != nil)
		assert.Equal(testElement.expectNilVal, cert == nil)
	}
}

package providers

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests"
)

func TestSynchronizeCertificate(t *testing.T) {
	assert := tassert.New(t)
	kubeClient := fake.NewSimpleClientset()

	wg := sync.WaitGroup{}
	wg.Add(10)
	certResults := make([]*certificate.Certificate, 10)

	// Test synchronization, expect all routines end up with the same cert
	for i := 0; i < 10; i++ {
		go func(num int) {
			defer wg.Done()

			resCert, err := GetOrCreateTresorCA("test", "test", kubeClient)
			assert.NoError(err)

			certResults[num] = resCert
		}(i)
	}
	wg.Wait()

	// Verifiy all of them loaded the exact same cert
	for i := 0; i < 9; i++ {
		assert.Equal(certResults[i], certResults[i+1])
	}
}

func TestGetCertificateFromKubernetes(t *testing.T) {
	assert := tassert.New(t)

	certPEM, err := tests.GetPEMCert()
	assert.NoError(err)
	keyPEM, err := tests.GetPEMPrivateKey()
	assert.NoError(err)

	ns := uuid.New().String()
	secretName := uuid.New().String()

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
			// Error when Private Key is missing
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: ns,
				},
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey: certPEM,
				},
			},
			expectError:  true,
			expectNilVal: true,
		},
	}

	for _, testElement := range testCases {
		kubeClient := fake.NewSimpleClientset()

		if testElement.secret != nil {
			_, err = kubeClient.CoreV1().Secrets(ns).Create(context.Background(), testElement.secret, metav1.CreateOptions{})
			assert.NoError(err)
		}

		cert, err := GetCertFromKubernetes(ns, secretName, kubeClient)

		assert.Equal(testElement.expectError, err != nil)
		assert.Equal(testElement.expectNilVal, cert == nil)
	}
}

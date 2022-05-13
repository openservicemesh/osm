package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
)

func TestGetCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()

	testCases := []struct {
		name        string
		expectError bool

		// params
		kubeClient        kubernetes.Interface
		kubeConfig        *rest.Config
		cfg               configurator.Configurator
		providerNamespace string
		options           Options
		msgBroker         *messaging.Broker
	}{
		{
			name:              "tresor as the certificate manager",
			options:           TresorOptions{SecretName: "osm-ca-bundle"},
			providerNamespace: "osm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			expectError:       false,
		},
		{
			name:              "tresor with no secret",
			options:           TresorOptions{},
			providerNamespace: "osm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			expectError:       true,
		},
		{
			name:              "certManager as the certificate manager",
			kubeClient:        fake.NewSimpleClientset(),
			kubeConfig:        &rest.Config{},
			cfg:               mockConfigurator,
			providerNamespace: "osm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "test-kind", IssuerGroup: "test-group", SecretName: "test-secret"},
			expectError:       false,
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			if opts, ok := tc.options.(CertManagerOptions); ok {
				secret := corev1.Secret{Data: map[string][]byte{constants.KubernetesOpaqueSecretCAKey: []byte(certificates.SampleCertificatePEM)}}
				secret.Name = opts.SecretName
				_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Create(context.Background(), &secret, metav1.CreateOptions{})
				assert.Nil(err)
			}

			manager, err := NewCertificateManager(tc.kubeClient, tc.kubeConfig, tc.cfg, tc.providerNamespace, tc.options, tc.msgBroker)
			assert.Equal(tc.expectError, manager == nil)
			assert.Equal(tc.expectError, err != nil)

			if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
				_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
				assert.NoError(err)
			}
		})
	}
}

func TestGetHashiVaultOSMCertificateManager(t *testing.T) {
	generator := &MRCProviderGenerator{
		KeyBitSize: 2048,
	}

	opt := VaultOptions{
		VaultHost:  "vault.default.svc.cluster.local",
		VaultToken: "vault-token",
		VaultRole:  "role",
		VaultPort:  8200,
	}

	testCases := []struct {
		name          string
		vaultProtocol string
		expErr        bool
		errMsg        string // optional error message to validate against
	}{
		{
			name:          "Not a valid Vault protocol",
			vaultProtocol: "hi",
			expErr:        true,
		},
		{
			name:          "Valid protocol, but can't reach out to vault",
			vaultProtocol: "http",
			expErr:        true,
			errMsg:        "error getting Vault Root Certificate",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			opt.VaultProtocol = tc.vaultProtocol

			mrc := &v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-mrc",
					Namespace: "test-namespace",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: opt.AsProviderSpec(),
				},
			}

			_, err := generator.getHashiVaultOSMCertificateManager(mrc)
			if tc.expErr {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}

			if tc.errMsg != "" {
				assert.Contains(err.Error(), tc.errMsg)
			}
		})
	}
}

func TestGetCertManagerOSMCertificateManager(t *testing.T) {
	generator := &MRCProviderGenerator{
		kubeClient: fake.NewSimpleClientset(),
		kubeConfig: &rest.Config{},
		KeyBitSize: 2048,
	}

	opt := CertManagerOptions{
		IssuerName:  "test-name",
		IssuerKind:  "test-kind",
		IssuerGroup: "test-group",
		SecretName:  "test-secret",
	}

	mrc := &v1alpha2.MeshRootCertificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mrc",
			Namespace: "osm-system",
		},
		Spec: v1alpha2.MeshRootCertificateSpec{
			Provider: opt.AsProviderSpec(),
		},
	}

	testCases := []struct {
		name         string
		createSecret bool
		secret       corev1.Secret
		expErr       bool
	}{
		{
			name:         "No secret",
			createSecret: false,
			secret:       corev1.Secret{},
			expErr:       true,
		},
		{
			name:         "Doesn't have opaque key",
			createSecret: true,
			secret:       corev1.Secret{},
			expErr:       true,
		},
		{
			name:         "Failed to decode",
			createSecret: true,
			secret:       corev1.Secret{Data: map[string][]byte{constants.KubernetesOpaqueSecretCAKey: {}}},
			expErr:       true,
		},
		{
			name:         "Successfully get CertManager",
			createSecret: true,
			secret:       corev1.Secret{Data: map[string][]byte{constants.KubernetesOpaqueSecretCAKey: []byte(certificates.SampleCertificatePEM)}},
			expErr:       false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			if tc.createSecret {
				tc.secret.Name = opt.SecretName
				_, err := generator.kubeClient.CoreV1().Secrets(mrc.Namespace).Create(context.Background(), &tc.secret, metav1.CreateOptions{})
				assert.Nil(err)
			}

			_, err := generator.getCertManagerOSMCertificateManager(mrc)
			assert.Equal(tc.expErr, err != nil)

			if tc.createSecret {
				err := generator.kubeClient.CoreV1().Secrets(mrc.Namespace).Delete(context.Background(), opt.SecretName, metav1.DeleteOptions{})
				assert.Nil(err)
			}
		})
	}
}

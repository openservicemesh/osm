package providers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
)

func TestNewCertificateProvider(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	kubeConfig, _ := clientConfig.ClientConfig()
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()

	testCases := []struct {
		name           string
		tresorOpt      TresorOptions
		vaultOpt       VaultOptions
		certManagerOpt CertManagerOptions
		providerKind   Kind
		expErr         bool
	}{
		{
			name:           "Successfully create certManager and certDebugger",
			tresorOpt:      TresorOptions{},
			vaultOpt:       VaultOptions{},
			certManagerOpt: CertManagerOptions{},
			providerKind:   TresorKind,
			expErr:         false,
		},
		{
			name:           "Fail to validate Config",
			tresorOpt:      TresorOptions{},
			vaultOpt:       VaultOptions{},
			certManagerOpt: CertManagerOptions{},
			providerKind:   VaultKind,
			expErr:         true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)
			_, _, _, err := NewCertificateProvider(fakeClient, kubeConfig, mockConfigurator, tc.providerKind, "osm-system", "osm-ca-bundle", tc.tresorOpt, tc.vaultOpt, tc.certManagerOpt, nil)
			assert.Equal(tc.expErr, err != nil)
		})
	}
}

func TestGetCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()

	testCases := []struct {
		name        string
		util        *Config
		expectError bool
	}{
		{
			name: "tresor as the certificate manager",
			util: &Config{
				caBundleSecretName: "osm-ca-bundle",
				providerKind:       TresorKind,
				providerNamespace:  "osm-system",
				cfg:                mockConfigurator,
				kubeClient:         fake.NewSimpleClientset(),
			},
			expectError: false,
		},
		{
			name: "certManager as the certificate manager",
			util: &Config{
				kubeClient:         fake.NewSimpleClientset(),
				kubeConfig:         &rest.Config{},
				cfg:                mockConfigurator,
				providerKind:       CertManagerKind,
				providerNamespace:  "osm-system",
				caBundleSecretName: "",
				certManagerOptions: CertManagerOptions{IssuerName: "test-name", IssuerKind: "test-kind", IssuerGroup: "test-group"},
			},
			expectError: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			if tc.util.providerKind == CertManagerKind {
				secret := corev1.Secret{Data: map[string][]byte{constants.KubernetesOpaqueSecretCAKey: []byte(certificates.SampleCertificatePEM)}}
				_, err := tc.util.kubeClient.CoreV1().Secrets(tc.util.providerNamespace).Create(context.Background(), &secret, metav1.CreateOptions{})
				assert.Nil(err)
			}

			manager, _, err := tc.util.GetCertificateManager()
			assert.NotNil(manager)
			assert.Equal(tc.expectError, err != nil)

			if tc.util.providerKind == TresorKind {
				_, err := tc.util.kubeClient.CoreV1().Secrets(tc.util.providerNamespace).Get(context.TODO(), tc.util.caBundleSecretName, metav1.GetOptions{})
				assert.NoError(err)
			}
		})
	}
}

func TestSynchronizeCertificate(t *testing.T) {
	assert := tassert.New(t)
	kubeClient := fake.NewSimpleClientset()

	// Create some cert, using tresor's api for simplicity
	cert, err := tresor.NewCA("common-name", time.Hour, "test-country", "test-locality", "test-org")
	assert.NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(10)
	certResults := make([]*certificate.Certificate, 10)

	// Test synchronization, expect all routines end up with the same cert
	for i := 0; i < 10; i++ {
		go func(num int) {
			defer wg.Done()

			resCert, err := GetCertificateFromSecret("test", "test", cert, kubeClient)
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

func TestValidateCertManagerOptions(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName  string
		options   CertManagerOptions
		expectErr bool
	}{
		{
			testName: "Empty issuer",
			options: CertManagerOptions{
				IssuerName:  "",
				IssuerKind:  "test-kind",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty kind",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "",
				IssuerGroup: "test-group",
			},
			expectErr: true,
		},
		{
			testName: "Empty group",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "test-kind",
				IssuerGroup: "",
			},
			expectErr: true,
		},
		{
			testName: "Valid cert manager opts",
			options: CertManagerOptions{
				IssuerName:  "test-name",
				IssuerKind:  "test-kind",
				IssuerGroup: "test-group",
			},
			expectErr: false,
		},
	}

	for _, t := range testCases {
		err := ValidateCertManagerOptions(t.options)
		if t.expectErr {
			assert.Error(err, "test '%s' didn't error as expected", t.testName)
		} else {
			assert.NoError(err, "test '%s' didn't succeed as expected", t.testName)
		}
	}
}

func TestValidateVaultOptions(t *testing.T) {
	assert := tassert.New(t)

	testCases := []struct {
		testName  string
		options   VaultOptions
		expectErr bool
	}{
		{
			testName: "invalid proto",
			options: VaultOptions{
				VaultProtocol: "ftp",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty host",
			options: VaultOptions{
				VaultProtocol: "http",
				VaultHost:     "",
				VaultToken:    "vault-token",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty token",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "",
				VaultRole:     "vault-role",
			},
			expectErr: true,
		},
		{
			testName: "Empty role",
			options: VaultOptions{
				VaultProtocol: "http",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "",
			},
			expectErr: true,
		},
		{
			testName: "Empty role",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "",
			},
			expectErr: true,
		},
		{
			testName: "Valid config",
			options: VaultOptions{
				VaultProtocol: "https",
				VaultHost:     "vault-host",
				VaultToken:    "vault-token",
				VaultRole:     "role",
			},
			expectErr: false,
		},
	}

	for _, t := range testCases {
		err := ValidateVaultOptions(t.options)
		if t.expectErr {
			assert.Error(err, "test '%s' didn't error as expected", t.testName)
		} else {
			assert.NoError(err, "test '%s' didn't succeed as expected", t.testName)
		}
	}
}

func TestGetHashiVaultOSMCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()
	config := Config{
		kubeConfig:         &rest.Config{},
		caBundleSecretName: "",
		providerKind:       VaultKind,
		providerNamespace:  "osm-system",
		cfg:                mockConfigurator,
		kubeClient:         fake.NewSimpleClientset(),
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
	}{
		{
			name:          "Not a valid Vault protocol",
			vaultProtocol: "hi",
			expErr:        true,
		},
		{
			name:          "Error instantiating Vault as CertManager",
			vaultProtocol: "http",
			expErr:        true,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Testing test case %d: %s", i, tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			opt.VaultProtocol = tc.vaultProtocol
			_, _, err := config.getHashiVaultOSMCertificateManager(opt)
			assert.Equal(tc.expErr, err != nil)
		})
	}
}

func TestGetCertManagerOSMCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()
	config := Config{
		kubeConfig:         &rest.Config{},
		caBundleSecretName: "",
		providerKind:       CertManagerKind,
		providerNamespace:  "osm-system",
		cfg:                mockConfigurator,
		kubeClient:         fake.NewSimpleClientset(),
	}
	opt := CertManagerOptions{
		IssuerName:  "test-name",
		IssuerKind:  "test-kind",
		IssuerGroup: "test-group",
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
				_, err := config.kubeClient.CoreV1().Secrets(config.providerNamespace).Create(context.Background(), &tc.secret, metav1.CreateOptions{})
				assert.Nil(err)
			}

			_, _, err := config.getCertManagerOSMCertificateManager(opt)
			assert.Equal(tc.expErr, err != nil)

			if tc.createSecret {
				err := config.kubeClient.CoreV1().Secrets(config.providerNamespace).Delete(context.Background(), "", metav1.DeleteOptions{})
				assert.Nil(err)
			}
		})
	}
}

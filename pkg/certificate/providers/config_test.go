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
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openservicemesh/osm/pkg/certificate/castorage/k8s"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/tests/certificates"
)

func TestGenerateCertificateManager(t *testing.T) {
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
			_, _, _, err := GenerateCertificateManager(fakeClient, kubeConfig, mockConfigurator, tc.providerKind, "osm-system", "osm-ca-bundle", tc.tresorOpt, tc.vaultOpt, tc.certManagerOpt, nil)
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

			// first setup the storage engine
			tc.util.caStorage = k8s.NewCASecretClient(tc.util.kubeClient, tc.util.caBundleSecretName, tc.util.providerNamespace, "")

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
	kubeClient := fake.NewSimpleClientset()
	config := Config{
		kubeConfig:         &rest.Config{},
		caBundleSecretName: "",
		providerKind:       CertManagerKind,
		providerNamespace:  "osm-system",
		cfg:                mockConfigurator,
		kubeClient:         kubeClient,
		caStorage:          k8s.NewCASecretClient(kubeClient, "", "osm-system", ""),
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

package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/messaging"
)

func TestGetCertificateManager(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockConfigurator := configurator.NewMockConfigurator(mockCtrl)

	mockConfigurator.EXPECT().IsDebugServerEnabled().Return(false).AnyTimes()
	mockConfigurator.EXPECT().GetCertKeyBitSize().Return(2048).AnyTimes()
	mockConfigurator.EXPECT().GetServiceCertValidityPeriod().Return(1 * time.Hour).AnyTimes()
	type testCase struct {
		name        string
		expectError bool

		// params
		kubeClient             kubernetes.Interface
		configClient           configClientset.Interface
		restConfig             *rest.Config
		cfg                    configurator.Configurator
		providerNamespace      string
		options                Options
		msgBroker              *messaging.Broker
		informerCollectionFunc func(testCase) (*informers.InformerCollection, error)
	}
	testCases := []testCase{
		{
			name:              "tresor as the certificate manager",
			options:           TresorOptions{SecretName: "osm-ca-bundle"},
			providerNamespace: "osm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "osm-ca-bundle",
									Namespace: "osm-system",
								},
							},
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}

				return ic, nil
			},
		},
		{
			name:              "tresor with no secret",
			options:           TresorOptions{},
			providerNamespace: "osm-system",
			cfg:               mockConfigurator,
			kubeClient:        fake.NewSimpleClientset(),
			expectError:       true,
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Tresor: &v1alpha2.TresorProviderSpec{
							CA: v1alpha2.TresorCASpec{
								SecretRef: v1.SecretReference{
									Name:      "",
									Namespace: "",
								},
							},
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
		{
			name:              "certManager as the certificate manager",
			kubeClient:        fake.NewSimpleClientset(),
			restConfig:        &rest.Config{},
			cfg:               mockConfigurator,
			providerNamespace: "osm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						CertManager: &v1alpha2.CertManagerProviderSpec{
							IssuerName:  "test-name",
							IssuerKind:  "ClusterIssuer",
							IssuerGroup: "cert-manager.io",
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			kubeClient:  fake.NewSimpleClientset(),
			expectError: true,
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Vault: &v1alpha2.VaultProviderSpec{
							Host:     "",
							Port:     0,
							Role:     "",
							Protocol: "",
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
		{
			name: "Valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
				VaultToken:    "vault-token",
			},
			cfg:        mockConfigurator,
			kubeClient: fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Vault: &v1alpha2.VaultProviderSpec{
							Host:     "vault.default.svs.cluster.local",
							Port:     8200,
							Role:     "role",
							Protocol: "http",
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
		{
			name: "Valid Vault protocol using vault secret defined in MRC",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
			},
			kubeClient: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vault-token",
					Namespace: "osm-system",
				},
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			}),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Vault: &v1alpha2.VaultProviderSpec{
							Host:     "vault.default.svc.cluster.local",
							Role:     "role",
							Port:     8200,
							Protocol: "http",
							Token: v1alpha2.VaultTokenSpec{
								SecretKeyRef: v1alpha2.SecretKeyReferenceSpec{
									Name:      "vault-token",
									Namespace: "osm-system",
									Key:       "token",
								},
							},
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
			cfg: mockConfigurator,
		},
		{
			name: "Not a valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "hi",
			},
			expectError: true,
			cfg:         mockConfigurator,
			kubeClient:  fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						Vault: &v1alpha2.VaultProviderSpec{
							Host:     "vault.default.svs.cluster.local",
							Port:     8200,
							Role:     "role",
							Protocol: "hi",
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
			cfg:        mockConfigurator,
			kubeClient: fake.NewSimpleClientset(),
			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: v1alpha2.MeshRootCertificateSpec{
					Provider: v1alpha2.ProviderSpec{
						CertManager: &v1alpha2.CertManagerProviderSpec{
							IssuerName:  "",
							IssuerKind:  "test-kind",
							IssuerGroup: "cert-manager.io",
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
				},
			}),
			expectError: true,
			informerCollectionFunc: func(tc testCase) (*informers.InformerCollection, error) {
				ic, err := informers.NewInformerCollection("osm", nil, informers.WithKubeClient(tc.kubeClient), informers.WithConfigClient(tc.configClient, "", "osm-system"))
				if err != nil {
					return nil, err
				}
				return ic, nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			oldCA := getCA
			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
				return pem.RootCertificate("id2"), nil
			}

			defer func() {
				getCA = oldCA
			}()

			ic, err := tc.informerCollectionFunc(tc)
			require.NoError(t, err)

			manager, err := NewCertificateManager(context.Background(), tc.kubeClient, tc.restConfig, tc.cfg, tc.providerNamespace, tc.options, tc.msgBroker, ic, 1*time.Hour)
			log.Info().Msgf("hello %s", err)
			if tc.expectError {
				assert.Empty(manager)
				assert.Error(err)
			} else {
				assert.NotEmpty(manager)
				assert.NoError(err)
			}

			if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
				_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
				assert.NoError(err)
			}
		})
	}
}

func TestGetHashiVaultOSMToken(t *testing.T) {
	validVaultTokenSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "osm-system",
			Name:      "osm-vault-token",
		},
		Data: map[string][]byte{
			"token": []byte("token"),
		},
	}

	invalidVaultTokenSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "osm-system",
			Name:      "osm-vault-token",
		},
		Data: map[string][]byte{
			"noop": []byte("noop"),
		},
	}

	testCases := []struct {
		name         string
		secretKeyRef *v1alpha2.SecretKeyReferenceSpec
		kubeClient   kubernetes.Interface
		expectError  bool
	}{
		{
			name: "No Vault token secret",
			secretKeyRef: &v1alpha2.SecretKeyReferenceSpec{
				Name:      "osm-vault-token",
				Namespace: "osm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset(),
			expectError: true,
		},
		{
			name: "Invalid Vault token secret",
			secretKeyRef: &v1alpha2.SecretKeyReferenceSpec{
				Name:      "osm-vault-token",
				Namespace: "osm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset([]runtime.Object{invalidVaultTokenSecret}...),
			expectError: true,
		},
		{
			name: "Valid Vault token secret",
			secretKeyRef: &v1alpha2.SecretKeyReferenceSpec{
				Name:      "osm-vault-token",
				Namespace: "osm-system",
				Key:       "token",
			},
			kubeClient:  fake.NewSimpleClientset([]runtime.Object{validVaultTokenSecret}...),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			token, err := getHashiVaultOSMToken(tc.secretKeyRef, tc.kubeClient)
			if tc.expectError {
				assert.Empty(token)
				assert.Error(err)
			} else {
				assert.NotEmpty(token)
				assert.NoError(err)
			}
		})
	}
}

package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	tassert "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/compute/kube"
	"github.com/openservicemesh/osm/pkg/constants"
	configClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned"
	fakeConfigClientset "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s"
	"github.com/openservicemesh/osm/pkg/messaging"
	"github.com/openservicemesh/osm/pkg/models"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/pem"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
)

func TestGetCertificateManager(t *testing.T) {
	cert, _ := tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

	type testCase struct {
		name        string
		expectError bool
		// params
		secret            *models.Secret
		restConfig        *rest.Config
		providerNamespace string
		options           Options
	}
	testCases := []testCase{
		{
			name: "tresor as the certificate manager",
			secret: &models.Secret{
				Name:      "osm-ca-bundle",
				Namespace: "osm-system",
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
				},
			},
			options:           TresorOptions{SecretName: "osm-ca-bundle"},
			providerNamespace: "osm-system",
		},
		{
			name:              "tresor with no secret",
			secret:            nil,
			options:           TresorOptions{},
			providerNamespace: "osm-system",
			expectError:       true,
		},
		{
			name:              "certManager as the certificate manager",
			restConfig:        &rest.Config{},
			providerNamespace: "osm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			expectError: true,
		},
		{
			name: "Valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
			},
		},
		{
			name: "Valid Vault protocol using vault secret",
			options: VaultOptions{
				VaultHost:                 "vault.default.svc.cluster.local",
				VaultRole:                 "role",
				VaultPort:                 8200,
				VaultProtocol:             "http",
				VaultTokenSecretName:      "secret",
				VaultTokenSecretKey:       "token",
				VaultTokenSecretNamespace: "osm-system",
			},
			secret: &models.Secret{
				Name:      "secret",
				Namespace: "osm-system",
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			},
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
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			mockCtrl := gomock.NewController(t)
			computeMock := compute.NewMockInterface(mockCtrl)
			computeMock.EXPECT().GetMeshConfig().AnyTimes()
			computeMock.EXPECT().CreateSecret(gomock.Any()).Return(nil).AnyTimes()
			if opt, ok := tc.options.(TresorOptions); ok {
				computeMock.EXPECT().GetSecret(opt.SecretName, tc.providerNamespace).Return(tc.secret).AnyTimes()
			} else if opt, ok := tc.options.(VaultOptions); ok {
				computeMock.EXPECT().GetSecret(opt.VaultTokenSecretName, opt.VaultTokenSecretNamespace).Return(tc.secret).AnyTimes()
			}

			oldCA := getCA
			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
				return pem.RootCertificate("id2"), nil
			}

			defer func() {
				getCA = oldCA
			}()

			manager, err := NewCertificateManager(context.Background(), tc.restConfig, tc.providerNamespace, tc.options, computeMock, 1*time.Hour, "cluster.local")
			if tc.expectError {
				assert.Empty(manager)
				assert.Error(err)
			} else {
				assert.NotEmpty(manager)
				assert.NoError(err)
			}
		})
	}
}

func TestGetCertificateManagerFromMRC(t *testing.T) {
	// cert, _ := tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

	type testCase struct {
		name        string
		expectError bool

		// params
		kubeClient        kubernetes.Interface
		secret            *models.Secret
		configClient      configClientset.Interface
		restConfig        *rest.Config
		providerNamespace string
		options           Options
	}
	testCases := []testCase{
		// {
		// 	name:              "tresor as the certificate manager",
		// 	expectError:       false,
		// 	options:           TresorOptions{SecretName: "osm-ca-bundle"},
		// 	kubeClient:        fake.NewSimpleClientset(),
		// 	providerNamespace: "osm-system",
		// 	secret: &models.Secret{
		// 		Name:      "osm-ca-bundle",
		// 		Namespace: "osm-system",
		// 		Data: map[string][]byte{
		// 			constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
		// 			constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
		// 		},
		// 	},
		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "osm-mesh-root-certificate",
		// 			Namespace: "osm-system",
		// 		},
		// 		Spec: v1alpha2.MeshRootCertificateSpec{
		// 			TrustDomain: "cluster.local",
		// 			Intent:      constants.MRCIntentPassive,
		// 			Provider: v1alpha2.ProviderSpec{
		// 				Tresor: &v1alpha2.TresorProviderSpec{
		// 					CA: v1alpha2.TresorCASpec{
		// 						SecretRef: v1.SecretReference{
		// 							Name:      "osm-ca-bundle",
		// 							Namespace: "osm-system",
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 		Status: v1alpha2.MeshRootCertificateStatus{
		// 			State: constants.MRCStateActive,
		// 			// unspecified component status will be unknown.
		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
		// 				{
		// 					Type:   constants.MRCConditionTypeReady,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeAccepted,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 			},
		// 		},
		// 	}),
		// },
		// {
		// 	name:              "tresor with no secret",
		// 	options:           TresorOptions{},
		// 	providerNamespace: "osm-system",
		// 	kubeClient:        fake.NewSimpleClientset(),
		// 	expectError:       true,
		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "osm-mesh-root-certificate",
		// 			Namespace: "osm-system",
		// 		},
		// 		Spec: v1alpha2.MeshRootCertificateSpec{
		// 			Provider: v1alpha2.ProviderSpec{
		// 				Tresor: &v1alpha2.TresorProviderSpec{
		// 					CA: v1alpha2.TresorCASpec{
		// 						SecretRef: v1.SecretReference{
		// 							Name:      "",
		// 							Namespace: "",
		// 						},
		// 					},
		// 				},
		// 			},
		// 		},
		// 		Status: v1alpha2.MeshRootCertificateStatus{
		// 			State: constants.MRCStateActive,
		// 			// unspecified component status will be unknown.
		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
		// 				{
		// 					Type:   constants.MRCConditionTypeReady,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeAccepted,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 			},
		// 		},
		// 	}),
		// },
		// {
		// 	name:              "certManager as the certificate manager",
		// 	kubeClient:        fake.NewSimpleClientset(),
		// 	restConfig:        &rest.Config{},
		// 	providerNamespace: "osm-system",
		// 	options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "osm-mesh-root-certificate",
		// 			Namespace: "osm-system",
		// 		},
		// 		Spec: v1alpha2.MeshRootCertificateSpec{
		// 			Provider: v1alpha2.ProviderSpec{
		// 				CertManager: &v1alpha2.CertManagerProviderSpec{
		// 					IssuerName:  "test-name",
		// 					IssuerKind:  "ClusterIssuer",
		// 					IssuerGroup: "cert-manager.io",
		// 				},
		// 			},
		// 		},
		// 		Status: v1alpha2.MeshRootCertificateStatus{
		// 			State: constants.MRCStateActive,
		// 			// unspecified component status will be unknown.
		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
		// 				{
		// 					Type:   constants.MRCConditionTypeReady,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeAccepted,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 			},
		// 		},
		// 	}),
		// },
		// {
		// 	name:        "Fail to validate Config",
		// 	options:     VaultOptions{},
		// 	kubeClient:  fake.NewSimpleClientset(),
		// 	expectError: true,
		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "osm-mesh-root-certificate",
		// 			Namespace: "osm-system",
		// 		},
		// 		Spec: v1alpha2.MeshRootCertificateSpec{
		// 			Provider: v1alpha2.ProviderSpec{
		// 				Vault: &v1alpha2.VaultProviderSpec{
		// 					Host:     "",
		// 					Port:     0,
		// 					Role:     "",
		// 					Protocol: "",
		// 				},
		// 			},
		// 		},
		// 		Status: v1alpha2.MeshRootCertificateStatus{
		// 			State: constants.MRCStateActive,
		// 			// unspecified component status will be unknown.
		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
		// 				{
		// 					Type:   constants.MRCConditionTypeReady,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeAccepted,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 			},
		// 		},
		// 	}),
		// },
		// {
		// 	name: "Valid Vault protocol",
		// 	options: VaultOptions{
		// 		VaultHost:     "vault.default.svc.cluster.local",
		// 		VaultRole:     "role",
		// 		VaultPort:     8200,
		// 		VaultProtocol: "http",
		// 		VaultToken:    "vault-token",
		// 	},
		// 	kubeClient: fake.NewSimpleClientset(),
		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
		// 		ObjectMeta: metav1.ObjectMeta{
		// 			Name:      "osm-mesh-root-certificate",
		// 			Namespace: "osm-system",
		// 		},
		// 		Spec: v1alpha2.MeshRootCertificateSpec{
		// 			Provider: v1alpha2.ProviderSpec{
		// 				Vault: &v1alpha2.VaultProviderSpec{
		// 					Host:     "vault.default.svs.cluster.local",
		// 					Port:     8200,
		// 					Role:     "role",
		// 					Protocol: "http",
		// 				},
		// 			},
		// 		},
		// 		Status: v1alpha2.MeshRootCertificateStatus{
		// 			State: constants.MRCStateActive,
		// 			// unspecified component status will be unknown.
		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
		// 				{
		// 					Type:   constants.MRCConditionTypeReady,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeAccepted,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollout,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeIssuingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 				{
		// 					Type:   constants.MRCConditionTypeValidatingRollback,
		// 					Status: constants.MRCConditionStatusUnknown,
		// 				},
		// 			},
		// 		},
		// 	}),
		// },
		{
			name:              "Valid Vault protocol using vault secret",
			providerNamespace: "osm-system",
			options: VaultOptions{
				VaultHost:                 "vault.default.svc.cluster.local",
				VaultRole:                 "role",
				VaultPort:                 8200,
				VaultProtocol:             "http",
				VaultTokenSecretName:      "secret",
				VaultTokenSecretKey:       "token",
				VaultTokenSecretNamespace: "osm-system",
			},
			kubeClient: fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "secret",
					Namespace: "osm-system",
				},
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			}),
			secret: &models.Secret{
				Name:      "secret",
				Namespace: "osm-system",
				Data: map[string][]byte{
					"token": []byte("secret"),
				},
			},
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
									Name:      "secret",
									Namespace: "osm-system",
									Key:       "token",
								},
							},
						},
					},
				},
				Status: v1alpha2.MeshRootCertificateStatus{
					State: constants.MRCStateActive,
					// unspecified component status will be unknown.
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeReady,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
					},
				},
			}),
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
					// unspecified component status will be unknown.
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeReady,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
					},
				},
			}),
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
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
					// unspecified component status will be unknown.
					Conditions: []v1alpha2.MeshRootCertificateCondition{
						{
							Type:   constants.MRCConditionTypeReady,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeAccepted,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollout,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeIssuingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
						{
							Type:   constants.MRCConditionTypeValidatingRollback,
							Status: constants.MRCConditionStatusUnknown,
						},
					},
				},
			}),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			// mockCtrl := gomock.NewController(t)
			// computeMock := compute.NewMockInterface(mockCtrl)
			// if !tc.expectError {
			// 	computeMock.EXPECT().AddMeshRootCertificateEventHandler(gomock.Any(), gomock.Any())
			// 	if opt, ok := tc.options.(TresorOptions); ok {
			// 		computeMock.EXPECT().GetSecret(opt.SecretName, tc.providerNamespace).Return(tc.secret)
			// 	}
			// }
			// computeMock.EXPECT().GetMeshConfig().AnyTimes()
			// computeMock.EXPECT().CreateSecret(gomock.Any()).Return(nil).AnyTimes()

			// if opt, ok := tc.options.(VaultOptions); ok {
			// 	computeMock.EXPECT().GetSecret(opt.VaultTokenSecretName, opt.VaultTokenSecretNamespace).Return(tc.secret).AnyTimes()
			// }
			oldCA := getCA
			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
				return pem.RootCertificate("id2"), nil
			}

			defer func() {
				getCA = oldCA
			}()

			stop := make(chan struct{})
			client, err := k8s.NewClient(tc.providerNamespace, "osm-mesh-config", messaging.NewBroker(stop),
				k8s.WithKubeClient(tc.kubeClient, "osm"),
				k8s.WithConfigClient(tc.configClient),
			)
			assert.Nil(err)
			computeClient := kube.NewClient(client)

			manager, err := NewCertificateManagerFromMRC(context.Background(), tc.restConfig, tc.providerNamespace, tc.options, computeClient, 1*time.Hour)
			if tc.expectError {
				assert.Empty(manager)
				assert.Error(err)
			} else {
				// assert.NotEmpty(manager)
				assert.NoError(err)
			}

			// if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
			// 	_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
			// 	assert.NoError(err)
			// }
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
	mockCtrl := gomock.NewController(t)
	computeMock := compute.NewMockInterface(mockCtrl)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)
			var secret *models.Secret
			if !tc.expectError {
				secret = &models.Secret{
					Name:      tc.secretKeyRef.Name,
					Namespace: tc.secretKeyRef.Namespace,
					Data:      map[string][]byte{tc.secretKeyRef.Key: {1}},
				}
			}

			computeMock.EXPECT().GetSecret(tc.secretKeyRef.Name, tc.secretKeyRef.Namespace).Return(secret)
			token, err := getHashiVaultOSMToken(tc.secretKeyRef, computeMock)
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

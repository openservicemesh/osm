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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/compute"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/models"
	"github.com/openservicemesh/osm/pkg/tests"
	"github.com/openservicemesh/osm/pkg/version"

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

// func TestGetCertificateManagerFromMRC(t *testing.T) {
// 	cert, _ := tresor.NewCA(constants.CertificationAuthorityCommonName, constants.CertificationAuthorityRootValidityPeriod, rootCertCountry, rootCertLocality, rootCertOrganization)

// 	type testCase struct {
// 		name        string
// 		expectError bool

// 		// params
// 		kubeClient        kubernetes.Interface
// 		secret            *models.Secret
// 		configClient      configClientset.Interface
// 		restConfig        *rest.Config
// 		providerNamespace string
// 		options           Options
// 	}
// 	testCases := []testCase{
// 		{
// 			name:              "tresor as the certificate manager",
// 			expectError:       false,
// 			options:           TresorOptions{SecretName: "osm-ca-bundle"},
// 			kubeClient:        fake.NewSimpleClientset(),
// 			providerNamespace: "osm-system",
// 			secret: &models.Secret{
// 				Name:      "osm-ca-bundle",
// 				Namespace: "osm-system",
// 				Data: map[string][]byte{
// 					constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
// 					constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
// 				},
// 			},
// 			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "osm-mesh-root-certificate",
// 					Namespace: "osm-system",
// 				},
// 				Spec: v1alpha2.MeshRootCertificateSpec{
// 					TrustDomain: "cluster.local",
// 					Intent:      constants.MRCIntentPassive,
// 					Provider: v1alpha2.ProviderSpec{
// 						Tresor: &v1alpha2.TresorProviderSpec{
// 							CA: v1alpha2.TresorCASpec{
// 								SecretRef: v1.SecretReference{
// 									Name:      "osm-ca-bundle",
// 									Namespace: "osm-system",
// 								},
// 							},
// 						},
// 					},
// 				},
// 				Status: v1alpha2.MeshRootCertificateStatus{
// 					State: constants.MRCStateActive,
// 					// unspecified component status will be unknown.
// 					Conditions: []v1alpha2.MeshRootCertificateCondition{
// 						{
// 							Type:   constants.MRCConditionTypeReady,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeAccepted,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeIssuingRollout,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeValidatingRollout,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeIssuingRollback,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeValidatingRollback,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 					},
// 				},
// 			}),
// 		},
// 		{
// 			name:              "tresor with no secret",
// 			options:           TresorOptions{},
// 			providerNamespace: "osm-system",
// 			kubeClient:        fake.NewSimpleClientset(),
// 			expectError:       true,
// 			configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "osm-mesh-root-certificate",
// 					Namespace: "osm-system",
// 				},
// 				Spec: v1alpha2.MeshRootCertificateSpec{
// 					Provider: v1alpha2.ProviderSpec{
// 						Tresor: &v1alpha2.TresorProviderSpec{
// 							CA: v1alpha2.TresorCASpec{
// 								SecretRef: v1.SecretReference{
// 									Name:      "",
// 									Namespace: "",
// 								},
// 							},
// 						},
// 					},
// 				},
// 				Status: v1alpha2.MeshRootCertificateStatus{
// 					State: constants.MRCStateActive,
// 					// unspecified component status will be unknown.
// 					Conditions: []v1alpha2.MeshRootCertificateCondition{
// 						{
// 							Type:   constants.MRCConditionTypeReady,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeAccepted,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeIssuingRollout,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeValidatingRollout,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeIssuingRollback,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 						{
// 							Type:   constants.MRCConditionTypeValidatingRollback,
// 							Status: constants.MRCConditionStatusUnknown,
// 						},
// 					},
// 				},
// 			}),
// 		},
// 		// {
// 		// 	name:              "certManager as the certificate manager",
// 		// 	kubeClient:        fake.NewSimpleClientset(),
// 		// 	restConfig:        &rest.Config{},
// 		// 	providerNamespace: "osm-system",
// 		// 	options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				CertManager: &v1alpha2.CertManagerProviderSpec{
// 		// 					IssuerName:  "test-name",
// 		// 					IssuerKind:  "ClusterIssuer",
// 		// 					IssuerGroup: "cert-manager.io",
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// },
// 		// {
// 		// 	name:        "Fail to validate Config",
// 		// 	options:     VaultOptions{},
// 		// 	kubeClient:  fake.NewSimpleClientset(),
// 		// 	expectError: true,
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				Vault: &v1alpha2.VaultProviderSpec{
// 		// 					Host:     "",
// 		// 					Port:     0,
// 		// 					Role:     "",
// 		// 					Protocol: "",
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// },
// 		// {
// 		// 	name: "Valid Vault protocol",
// 		// 	options: VaultOptions{
// 		// 		VaultHost:     "vault.default.svc.cluster.local",
// 		// 		VaultRole:     "role",
// 		// 		VaultPort:     8200,
// 		// 		VaultProtocol: "http",
// 		// 		VaultToken:    "vault-token",
// 		// 	},
// 		// 	kubeClient: fake.NewSimpleClientset(),
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				Vault: &v1alpha2.VaultProviderSpec{
// 		// 					Host:     "vault.default.svs.cluster.local",
// 		// 					Port:     8200,
// 		// 					Role:     "role",
// 		// 					Protocol: "http",
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// },
// 		// {
// 		// 	name: "Valid Vault protocol using vault secret",
// 		// 	options: VaultOptions{
// 		// 		VaultHost:                 "vault.default.svc.cluster.local",
// 		// 		VaultRole:                 "role",
// 		// 		VaultPort:                 8200,
// 		// 		VaultProtocol:             "http",
// 		// 		VaultTokenSecretName:      "secret",
// 		// 		VaultTokenSecretKey:       "token",
// 		// 		VaultTokenSecretNamespace: "osm-system",
// 		// 	},
// 		// 	kubeClient: fake.NewSimpleClientset(&v1.Secret{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "secret",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Data: map[string][]byte{
// 		// 			"token": []byte("secret"),
// 		// 		},
// 		// 	}),
// 		// 	secret: &models.Secret{
// 		// 		Name:      "secret",
// 		// 		Namespace: "osm-system",
// 		// 		Data: map[string][]byte{
// 		// 			"token": []byte("secret"),
// 		// 		},
// 		// 	},
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				Vault: &v1alpha2.VaultProviderSpec{
// 		// 					Host:     "vault.default.svc.cluster.local",
// 		// 					Role:     "role",
// 		// 					Port:     8200,
// 		// 					Protocol: "http",
// 		// 					Token: v1alpha2.VaultTokenSpec{
// 		// 						SecretKeyRef: v1alpha2.SecretKeyReferenceSpec{
// 		// 							Name:      "secret",
// 		// 							Namespace: "osm-system",
// 		// 							Key:       "token",
// 		// 						},
// 		// 					},
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// },
// 		// {
// 		// 	name: "Not a valid Vault protocol",
// 		// 	options: VaultOptions{
// 		// 		VaultHost:     "vault.default.svc.cluster.local",
// 		// 		VaultToken:    "vault-token",
// 		// 		VaultRole:     "role",
// 		// 		VaultPort:     8200,
// 		// 		VaultProtocol: "hi",
// 		// 	},
// 		// 	expectError: true,
// 		// 	kubeClient:  fake.NewSimpleClientset(),
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				Vault: &v1alpha2.VaultProviderSpec{
// 		// 					Host:     "vault.default.svs.cluster.local",
// 		// 					Port:     8200,
// 		// 					Role:     "role",
// 		// 					Protocol: "hi",
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// },
// 		// {
// 		// 	name: "Invalid cert manager options",
// 		// 	options: CertManagerOptions{
// 		// 		IssuerKind:  "test-kind",
// 		// 		IssuerGroup: "cert-manager.io",
// 		// 	},
// 		// 	kubeClient: fake.NewSimpleClientset(),
// 		// 	configClient: fakeConfigClientset.NewSimpleClientset(&v1alpha2.MeshRootCertificate{
// 		// 		ObjectMeta: metav1.ObjectMeta{
// 		// 			Name:      "osm-mesh-root-certificate",
// 		// 			Namespace: "osm-system",
// 		// 		},
// 		// 		Spec: v1alpha2.MeshRootCertificateSpec{
// 		// 			Provider: v1alpha2.ProviderSpec{
// 		// 				CertManager: &v1alpha2.CertManagerProviderSpec{
// 		// 					IssuerName:  "",
// 		// 					IssuerKind:  "test-kind",
// 		// 					IssuerGroup: "cert-manager.io",
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 		Status: v1alpha2.MeshRootCertificateStatus{
// 		// 			State: constants.MRCStateActive,
// 		// 			// unspecified component status will be unknown.
// 		// 			Conditions: []v1alpha2.MeshRootCertificateCondition{
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeReady,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeAccepted,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollout,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeIssuingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 				{
// 		// 					Type:   constants.MRCConditionTypeValidatingRollback,
// 		// 					Status: constants.MRCConditionStatusUnknown,
// 		// 				},
// 		// 			},
// 		// 		},
// 		// 	}),
// 		// 	expectError: true,
// 		// },
// 	}

// 	for _, tc := range testCases {
// 		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
// 			assert := tassert.New(t)
// 			mockCtrl := gomock.NewController(t)
// 			computeMock := compute.NewMockInterface(mockCtrl)
// 			if !tc.expectError {
// 				computeMock.EXPECT().AddMeshRootCertificateEventHandler(gomock.Any())
// 				if opt, ok := tc.options.(TresorOptions); ok {
// 					computeMock.EXPECT().GetSecret(opt.SecretName, tc.providerNamespace).Return(tc.secret)
// 				}
// 			}
// 			computeMock.EXPECT().GetMeshConfig().AnyTimes()
// 			computeMock.EXPECT().CreateSecret(gomock.Any()).Return(nil).AnyTimes()

// 			if opt, ok := tc.options.(VaultOptions); ok {
// 				computeMock.EXPECT().GetSecret(opt.VaultTokenSecretName, opt.VaultTokenSecretNamespace).Return(tc.secret).AnyTimes()
// 			}
// 			oldCA := getCA
// 			getCA = func(i certificate.Issuer) (pem.RootCertificate, error) {
// 				return pem.RootCertificate("id2"), nil
// 			}

// 			defer func() {
// 				getCA = oldCA
// 			}()

// 			// stop := make(chan struct{})
// 			// client, err := k8s.NewClient(tc.providerNamespace, "osm-mesh-config", messaging.NewBroker(stop),
// 			// 	k8s.WithKubeClient(tc.kubeClient, "osm"),
// 			// 	k8s.WithConfigClient(tc.configClient),
// 			// )
// 			// assert.Nil(err)
// 			// computeClient := kube.NewClient(client)

// 			manager, err := NewCertificateManagerFromMRC(context.Background(), tc.restConfig, tc.providerNamespace, tc.options, computeMock, 1*time.Hour)
// 			if tc.expectError {
// 				assert.Empty(manager)
// 				assert.Error(err)
// 			} else {
// 				// assert.NotEmpty(manager)
// 				assert.NoError(err)
// 			}
// 			context.Background().Done()

// 			// if opt, ok := tc.options.(TresorOptions); ok && !tc.expectError {
// 			// 	_, err := tc.kubeClient.CoreV1().Secrets(tc.providerNamespace).Get(context.TODO(), opt.SecretName, metav1.GetOptions{})
// 			// 	assert.NoError(err)
// 			// }
// 		})
// 	}
// }

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

func TestGetCertificateFromKubernetes(t *testing.T) {
	assert := tassert.New(t)

	certPEM, err := tests.GetPEMCert()
	assert.NoError(err)
	keyPEM, err := tests.GetPEMPrivateKey()
	assert.NoError(err)

	ns := uuid.New().String()
	secretName := uuid.New().String()

	testCases := []struct {
		secret       *models.Secret
		expectError  bool
		expectNilVal bool
	}{
		{
			// Valid cert, valid test
			secret: &models.Secret{
				Name:      secretName,
				Namespace: ns,
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
			secret: &models.Secret{
				Name:      secretName,
				Namespace: ns,
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretRootPrivateKeyKey: keyPEM,
				},
			},
			expectError:  true,
			expectNilVal: true,
		},
		{
			// Error when Private Key is missing
			secret: &models.Secret{
				Name:      secretName,
				Namespace: ns,
				Data: map[string][]byte{
					constants.KubernetesOpaqueSecretCAKey: certPEM,
				},
			},
			expectError:  true,
			expectNilVal: true,
		},
	}

	for _, tc := range testCases {
		mockInterface := compute.NewMockInterface(gomock.NewController(t))
		mockInterface.EXPECT().GetSecret(secretName, ns).Return(tc.secret)

		cert, err := getCertFromKubernetes(ns, secretName, mockInterface)

		assert.Equal(tc.expectError, err != nil)
		assert.Equal(tc.expectNilVal, cert == nil)
	}
}

func TestSynchronizeCertificate(t *testing.T) {
	assert := tassert.New(t)
	routineNum := 10
	// Create some cert, using tresor's api for simplicity
	cert, err := tresor.NewCA("common-name", time.Hour, "test-country", "test-locality", "test-org")
	assert.NoError(err)

	secret := &models.Secret{
		Name:      "test",
		Namespace: "test",
		Labels: map[string]string{
			constants.OSMAppNameLabelKey:    constants.OSMAppNameLabelValue,
			constants.OSMAppVersionLabelKey: version.Version,
		},
		Data: map[string][]byte{
			constants.KubernetesOpaqueSecretCAKey:             cert.GetCertificateChain(),
			constants.KubernetesOpaqueSecretRootPrivateKeyKey: cert.GetPrivateKey(),
		},
	}

	mockInterface := compute.NewMockInterface(gomock.NewController(t))
	mockInterface.EXPECT().CreateSecret(secret).Return(nil).MaxTimes(routineNum)
	mockInterface.EXPECT().GetSecret("test", "test").Return(secret).MaxTimes(routineNum)

	wg := sync.WaitGroup{}
	wg.Add(routineNum)
	certResults := make([]*certificate.Certificate, routineNum)

	// Test synchronization, expect all routines end up with the same cert
	for i := 0; i < routineNum; i++ {
		go func(num int) {
			defer wg.Done()

			resCert, err := getCertificateFromSecret("test", "test", cert, mockInterface)
			assert.NoError(err)
			certResults[num] = resCert
		}(i)
	}
	wg.Wait()

	// Verifiy all of them loaded the exact same cert
	for i := 0; i < routineNum-1; i++ {
		assert.Equal(certResults[i], certResults[i+1])
	}
}

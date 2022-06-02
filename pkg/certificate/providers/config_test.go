package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	certmanager "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmfakeclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/fake"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/messaging"
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
		certmanagerClient bool
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
			certmanagerClient: true,
			cfg:               mockConfigurator,
			providerNamespace: "osm-system",
			options:           CertManagerOptions{IssuerName: "test-name", IssuerKind: "ClusterIssuer", IssuerGroup: "cert-manager.io"},
			expectError:       false,
		},
		{
			name:        "Fail to validate Config",
			options:     VaultOptions{},
			expectError: true,
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
			name: "Valid Vault protocol",
			options: VaultOptions{
				VaultHost:     "vault.default.svc.cluster.local",
				VaultToken:    "vault-token",
				VaultRole:     "role",
				VaultPort:     8200,
				VaultProtocol: "http",
			},
			cfg:         mockConfigurator,
			expectError: false,
		},
		{
			name: "Invalid cert manager options",
			options: CertManagerOptions{
				IssuerKind:  "test-kind",
				IssuerGroup: "cert-manager.io",
			},
			cfg:         mockConfigurator,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf(tc.name), func(t *testing.T) {
			assert := tassert.New(t)

			var objects []runtime.Object
			if opt, ok := tc.options.(CertManagerOptions); ok && tc.certmanagerClient {
				cmIssuer := &certmanager.ClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: opt.IssuerName,
					},
					Spec: certmanager.IssuerSpec{
						IssuerConfig: certmanager.IssuerConfig{
							SelfSigned: &certmanager.SelfSignedIssuer{},
						},
					},
				}
				objects = append(objects, cmIssuer)
			}
			cmClient := cmfakeclient.NewSimpleClientset(objects...)

			manager, err := NewCertificateManager(tc.kubeClient, cmClient, tc.cfg, tc.providerNamespace, tc.options, tc.msgBroker)
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

package ingress

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	configv1alpha3 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha3"

	"github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/certificate/rotor"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	maxTimeToSubscribe           = 500 * time.Millisecond
	maxTimeForEventToBePublished = 2 * time.Second
	maxSecretPollTime            = 2 * time.Second
	secretPollInterval           = 25 * time.Millisecond
)

func TestProvisionIngressGatewayCert(t *testing.T) {
	testSecret := corev1.SecretReference{
		Name:      "gateway-cert",
		Namespace: "gateway-ns",
	}

	testCases := []struct {
		name                string
		meshConfig          configv1alpha3.MeshConfig
		expectSecretToExist bool
		expectErr           bool
	}{
		{
			name: "ingress gateway cert spec does not exist",
			meshConfig: configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: nil,
					},
				},
			},
			expectSecretToExist: false,
			expectErr:           false,
		},
		{
			name: "ingress gateway cert spec exists",
			meshConfig: configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: &configv1alpha3.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			expectSecretToExist: true,
			expectErr:           false,
		},
		{
			name: "ingress gateway cert spec has no SAN",
			meshConfig: configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: &configv1alpha3.IngressGatewayCertSpec{
							SubjectAltNames:  nil,
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			expectSecretToExist: false,
			expectErr:           true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			stop := make(chan struct{})
			defer close(stop)

			msgBroker := messaging.NewBroker(stop)

			fakeClient := fake.NewSimpleClientset()
			mockConfigurator := configurator.NewMockConfigurator(mockCtrl)
			fakeCertProvider := tresorFake.NewFake(msgBroker)

			c := client{
				kubeClient:   fakeClient,
				certProvider: fakeCertProvider,
				cfg:          mockConfigurator,
				msgBroker:    msgBroker,
			}

			mockConfigurator.EXPECT().GetMeshConfig().Return(tc.meshConfig).Times(1)

			stopChan := make(chan struct{})
			defer close(stopChan)

			err := c.provisionIngressGatewayCert(stopChan)
			a.Equal(tc.expectErr, err != nil)

			if tc.expectSecretToExist {
				a.Eventually(func() bool {
					_, err := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return err == nil
				}, maxSecretPollTime, secretPollInterval, "Secret not found, unexpected!")
			} else {
				a.Eventually(func() bool {
					_, err := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return err != nil
				}, maxSecretPollTime, secretPollInterval, "Secret found, unexpected!")
			}
		})
	}
}

func TestCreateAndStoreGatewayCert(t *testing.T) {
	testSecret := corev1.SecretReference{
		Name:      "gateway-cert",
		Namespace: "gateway-ns",
	}

	testCases := []struct {
		name      string
		certSpec  configv1alpha3.IngressGatewayCertSpec
		expectErr bool
	}{
		{
			name: "valid spec",
			certSpec: configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			expectErr: false,
		},
		{
			name: "invalid SAN",
			certSpec: configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  nil,
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			expectErr: true,
		},
		{
			name: "invalid validity duration",
			certSpec: configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "foobar",
				Secret:           testSecret,
			},
			expectErr: true,
		},
		{
			name: "invalid secret, name or namepace not specified",
			certSpec: configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret: corev1.SecretReference{
					Namespace: "foo",
					Name:      "",
				},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)

			fakeClient := fake.NewSimpleClientset()
			fakeCertProvider := tresorFake.NewFake(nil)

			c := client{
				kubeClient:   fakeClient,
				certProvider: fakeCertProvider,
			}

			err := c.createAndStoreGatewayCert(tc.certSpec)
			a.Equal(tc.expectErr, err != nil)
		})
	}
}

func TestHandleCertificateChange(t *testing.T) {
	testSecret := corev1.SecretReference{
		Name:      "gateway-cert",
		Namespace: "gateway-ns",
	}

	testCases := []struct {
		name                string
		previousCertSpec    *configv1alpha3.IngressGatewayCertSpec
		previousMeshConfig  *configv1alpha3.MeshConfig
		updatedMeshConfig   *configv1alpha3.MeshConfig
		stopChan            chan struct{}
		expectCertRotation  bool
		expectSecretToExist bool
	}{
		{
			name:               "setting spec when not previously set",
			previousCertSpec:   nil,
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: &configv1alpha3.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			stopChan:            make(chan struct{}),
			expectSecretToExist: true,
		},
		{
			name: "MeshConfig updated but certificate spec remains the same",
			previousCertSpec: &configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: &configv1alpha3.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			stopChan:            make(chan struct{}),
			expectSecretToExist: true,
		},
		{
			name: "MeshConfig and certificate spec updated",
			previousCertSpec: &configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: &configv1alpha3.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "2h",
							Secret:           testSecret,
						},
					},
				},
			},
			stopChan:            make(chan struct{}),
			expectSecretToExist: true,
		},
		{
			name: "Certificate spec is unset to remove certificate",
			previousCertSpec: &configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha3.MeshConfig{
				Spec: configv1alpha3.MeshConfigSpec{
					Certificate: configv1alpha3.CertificateSpec{
						IngressGateway: nil,
					},
				},
			},
			stopChan:            make(chan struct{}),
			expectSecretToExist: false,
		},
		{
			name: "Secret for rotated certificate is updated",
			previousCertSpec: &configv1alpha3.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "5s",
				Secret:           testSecret,
			},
			previousMeshConfig:  nil,
			updatedMeshConfig:   nil,
			stopChan:            make(chan struct{}),
			expectCertRotation:  true,
			expectSecretToExist: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			stop := make(chan struct{})
			defer close(stop)

			msgBroker := messaging.NewBroker(stop)

			fakeClient := fake.NewSimpleClientset()
			fakeCertManager := tresorFake.NewFake(msgBroker)

			c := client{
				kubeClient:   fakeClient,
				certProvider: fakeCertManager,
				msgBroker:    msgBroker,
			}

			go c.handleCertificateChange(tc.previousCertSpec, tc.stopChan)
			defer close(tc.stopChan)
			time.Sleep(maxTimeToSubscribe)

			// If a secret is supposed to exist, create it
			if tc.previousCertSpec != nil {
				err := c.createAndStoreGatewayCert(*tc.previousCertSpec)
				a.Nil(err)
			}

			if tc.updatedMeshConfig != nil {
				msgBroker.GetKubeEventPubSub().Pub(events.PubSubMessage{
					Kind:   announcements.MeshConfigUpdated,
					NewObj: tc.updatedMeshConfig,
					OldObj: tc.previousMeshConfig,
				}, announcements.MeshConfigUpdated.String())
				time.Sleep(maxTimeForEventToBePublished)
			}

			if !tc.expectSecretToExist {
				a.Eventually(func() bool {
					_, secretNotFoundErr := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})

					certCN := certificate.CommonName(tc.previousCertSpec.SubjectAltNames[0])
					_, certNotFoundErr := c.certProvider.GetCertificate(certCN)

					return secretNotFoundErr != nil && certNotFoundErr != nil
				}, maxSecretPollTime, secretPollInterval, "Secret found, unexpected!")
				return
			}

			if tc.updatedMeshConfig != nil {
				a.Eventually(func() bool {
					secret, err := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return err == nil && secretIsForSAN(secret, tc.updatedMeshConfig.Spec.Certificate.IngressGateway.SubjectAltNames[0])
				}, maxSecretPollTime, secretPollInterval, "Expected secret was not found")
			}

			if tc.expectCertRotation {
				// original secret
				originalSecret, err := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
				a.Nil(err)

				// Start the certificate rotor
				rotor.New(fakeCertManager).Start(5 * time.Second)

				a.Eventually(func() bool {
					rotatedSecret, err := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return err == nil && !reflect.DeepEqual(originalSecret, rotatedSecret)
				}, 10*time.Second /* additional time to account for rotation */, secretPollInterval, "Certificate was not rotated")
			}
		})
	}
}

func secretIsForSAN(secret *corev1.Secret, san string) bool {
	pemCert, ok := secret.Data["tls.crt"]
	if !ok {
		log.Error().Msg("PEM cert not found in secret")
		return false
	}

	pemKey, ok := secret.Data["tls.key"]
	if !ok {
		log.Error().Msg("PEM key not found in secret")
		return false
	}

	cert, err := tresor.NewCertificateFromPEM(pemCert, pemKey)
	if err != nil {
		log.Error().Err(err).Msg("Error getting certificate from PEM")
		return false
	}

	return cert.GetCommonName().String() == san
}

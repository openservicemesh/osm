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

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"

	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/k8s/events"
	"github.com/openservicemesh/osm/pkg/messaging"
)

const (
	maxTimeToSubscribe           = 500 * time.Millisecond
	maxTimeForEventToBePublished = 500 * time.Millisecond
	maxSecretPollTime            = 2 * time.Second
	secretPollInterval           = 10 * time.Millisecond
)

func TestProvisionIngressGatewayCert(t *testing.T) {
	testSecret := corev1.SecretReference{
		Name:      "gateway-cert",
		Namespace: "gateway-ns",
	}

	testCases := []struct {
		name                string
		previousCertSpec    *configv1alpha2.IngressGatewayCertSpec
		previousMeshConfig  *configv1alpha2.MeshConfig
		updatedMeshConfig   *configv1alpha2.MeshConfig
		expectCertRotation  bool
		expectSecretToExist bool
	}{
		{
			name:               "setting spec when not previously set",
			previousCertSpec:   nil,
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha2.MeshConfig{
				Spec: configv1alpha2.MeshConfigSpec{
					Certificate: configv1alpha2.CertificateSpec{
						IngressGateway: &configv1alpha2.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			expectSecretToExist: true,
		},
		{
			name: "MeshConfig updated but certificate spec remains the same",
			previousCertSpec: &configv1alpha2.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha2.MeshConfig{
				Spec: configv1alpha2.MeshConfigSpec{
					Certificate: configv1alpha2.CertificateSpec{
						IngressGateway: &configv1alpha2.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "1h",
							Secret:           testSecret,
						},
					},
				},
			},
			expectSecretToExist: true,
		},
		{
			name: "MeshConfig and certificate spec updated",
			previousCertSpec: &configv1alpha2.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.remote"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig: nil,
			updatedMeshConfig: &configv1alpha2.MeshConfig{
				Spec: configv1alpha2.MeshConfigSpec{
					Certificate: configv1alpha2.CertificateSpec{
						IngressGateway: &configv1alpha2.IngressGatewayCertSpec{
							SubjectAltNames:  []string{"foo.bar.cluster.local"},
							ValidityDuration: "2h",
							Secret:           testSecret,
						},
					},
				},
			},
			expectSecretToExist: true,
		},
		{
			name: "Certificate spec is unset to remove certificate",
			previousCertSpec: &configv1alpha2.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "1h",
				Secret:           testSecret,
			},
			previousMeshConfig:  nil,
			updatedMeshConfig:   &configv1alpha2.MeshConfig{},
			expectSecretToExist: false,
		},
		{
			name: "Secret for rotated certificate is updated",
			previousCertSpec: &configv1alpha2.IngressGatewayCertSpec{
				SubjectAltNames:  []string{"foo.bar.cluster.local"},
				ValidityDuration: "5ms",
				Secret:           testSecret,
			},
			previousMeshConfig:  nil,
			updatedMeshConfig:   nil,
			expectCertRotation:  true, // rotated due to short validity duration and sleeps.
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

			var err error
			validityDuration := 1 * time.Hour
			if tc.previousCertSpec != nil {
				validityDuration, err = time.ParseDuration(tc.previousCertSpec.ValidityDuration)
				a.NoError(err)
			}
			getCertValidityDuration := func() time.Duration {
				return validityDuration
			}

			fakeCertManager := tresorFake.NewFakeWithValidityDuration(getCertValidityDuration, 5*time.Second)

			fakeClient := fake.NewSimpleClientset()
			c := client{
				kubeClient:   fakeClient,
				certProvider: fakeCertManager,
				msgBroker:    msgBroker,
			}

			go c.provisionIngressGatewayCert(tc.previousCertSpec, stop)
			time.Sleep(maxTimeToSubscribe)

			var originalSecret *corev1.Secret
			if tc.expectCertRotation {
				a.Eventually(func() bool {
					originalSecret, err = fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return err == nil
				}, maxSecretPollTime, secretPollInterval, "Secret found, unexpected!")
			}

			if tc.updatedMeshConfig != nil {
				msgBroker.PublishKubeEvent(events.PubSubMessage{
					Kind:   events.MeshConfig,
					Type:   events.Updated,
					NewObj: tc.updatedMeshConfig,
					OldObj: tc.previousMeshConfig,
				})
				time.Sleep(maxTimeForEventToBePublished)
			}

			if !tc.expectSecretToExist {
				a.Eventually(func() bool {
					_, secretNotFoundErr := fakeClient.CoreV1().Secrets(testSecret.Namespace).Get(context.TODO(), testSecret.Name, metav1.GetOptions{})
					return secretNotFoundErr != nil
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

	cert, err := certificate.NewRootCertificateFromPEM(pemCert, pemKey)
	if err != nil {
		log.Error().Err(err).Msg("Error getting certificate from PEM")
		return false
	}
	return cert.GetCommonName().String() == san
}

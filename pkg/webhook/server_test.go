package webhook

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	configFake "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

func TestCertRotation(t *testing.T) {
	testCases := []struct {
		name              string
		rotations         int
		rotationsExpected int
		checkInterval     time.Duration
		waitForRotation   bool
	}{
		{
			name:              "should rotate certificate",
			rotations:         1,
			rotationsExpected: 1,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{
			name:              "should rotate certificate if CA changes multiple times",
			rotations:         2,
			rotationsExpected: 2,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{
			name:              "should not rotate certificate if no changes",
			rotations:         0,
			rotationsExpected: 0,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{
			name:              "should not rotate certificate if checkInterval is longer than rotation check interval",
			rotations:         1,
			rotationsExpected: 0,
			checkInterval:     1 * time.Hour,
			waitForRotation:   false,
		},
		{
			name:              "should not rotate certificate if checkInterval is longer than rotation check interval with multiple rotations",
			rotations:         2,
			rotationsExpected: 0,
			checkInterval:     1 * time.Hour,
			waitForRotation:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			mrc1 := &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate",
					Namespace: "osm-system",
				},
				Spec: configv1alpha2.MeshRootCertificateSpec{
					TrustDomain: "fake.example.com",
					Intent:      configv1alpha2.ActiveIntent,
				},
			}

			mrc2 := &configv1alpha2.MeshRootCertificate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "osm-mesh-root-certificate-2",
					Namespace: "osm-system",
				},
				Spec: configv1alpha2.MeshRootCertificateSpec{
					TrustDomain: "fake.example.com",
					Intent:      configv1alpha2.ActiveIntent,
				},
			}

			configClient := configFake.NewSimpleClientset([]runtime.Object{mrc1}...)
			mrcClient := tresorFake.NewFakeMRCWithConfig(configClient)
			cm := tresorFake.NewFakeWithMRCClient(mrcClient, tc.checkInterval)
			assert.NotNil(cm)

			count := 0
			wait := make(chan struct{})
			onCertChange := func(cert *certificate.Certificate) error {
				count++
				// This will always be called when server is initialized for
				// the first cert that is created to run the webserver
				if tc.waitForRotation || count == 1 {
					// send msg on a thread to not block
					go func() { wait <- struct{}{} }()
				}
				return nil
			}

			// create a cert before we start the server
			// Allows us to test there was a change in certs
			firstCert, _ := cm.IssueCertificate(certificate.ForCommonName("testhook.osm-system.svc"))

			server := NewServer("testhook", "osm-system", 6000, cm, nil, onCertChange)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := server.Run(ctx)
			assert.NoError(err)

			// wait for the first cert to be issued
			<-wait

			if tc.rotations >= 1 {
				// create mrc2 with passive intent
				_, err := configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Create(context.Background(), mrc2, metav1.CreateOptions{})
				assert.NoError(err)
				// trigger an MRCEvent to update the issuers and trigger a rotation
				mrcClient.NewCertEvent(mrc2.Name)
				if tc.waitForRotation {
					<-wait
				}
			}
			if tc.rotations == 2 {
				// set intent of mrc2 to active. This update will not trigger a rotation
				// do not wait on rotation
				mrc2.Spec.Intent = configv1alpha2.ActiveIntent
				_, err := configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Update(context.Background(), mrc2, metav1.UpdateOptions{})
				assert.NoError(err)

				// set intent of mrc1 to passive
				mrc1.Spec.Intent = configv1alpha2.PassiveIntent
				_, err = configClient.ConfigV1alpha2().MeshRootCertificates("osm-system").Update(context.Background(), mrc1, metav1.UpdateOptions{})
				// trigger an MRCEvent to update the issuers and trigger a rotation
				mrcClient.NewCertEvent(mrc1.Name)
				assert.NoError(err)
				if tc.waitForRotation {
					<-wait
				}
			}

			// get current cert
			certificates := cm.ListIssuedCertificates()
			currentCertificate := certificates[0]
			assert.NotNil(currentCertificate)

			// convert to the tls cert for comparison
			initialTLS, err := convertToTLSCert(firstCert)
			assert.Nil(err)

			currentTLS, err := convertToTLSCert(currentCertificate)
			assert.Nil(err)

			// each expected rotation + 1 for the original cert created
			totalCallsToOnRotation := tc.rotationsExpected + 1
			assert.Equal(totalCallsToOnRotation, count, "expected rotations %d got %d", totalCallsToOnRotation, count)

			if tc.rotationsExpected > 0 {
				// should not have same cert
				assert.NotEqual(initialTLS, server.cert)
				assert.Equal(currentTLS, server.cert)
			} else {
				// cert should be same as initial cert
				assert.Equal(initialTLS, server.cert)
				assert.Equal(currentTLS, server.cert)
			}
		})
	}
}

func convertToTLSCert(cert *certificate.Certificate) (tls.Certificate, error) {
	c, err := tls.X509KeyPair(cert.GetCertificateChain(), cert.GetPrivateKey())
	return c, err
}

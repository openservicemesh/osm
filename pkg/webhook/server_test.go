package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestCertRotation(t *testing.T) {
	testCases := []struct {
		name              string
		rotations         int
		rotationsExpected int
		checkInterval     time.Duration
		waitForRotation   bool
	}{
		{name: "should rotate certificate",
			rotations:         1,
			rotationsExpected: 1,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{name: "should rotate certificate if CA changes multiple times",
			rotations:         2,
			rotationsExpected: 2,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{name: "should not rotate certificate if no changes",
			rotations:         0,
			rotationsExpected: 0,
			checkInterval:     10 * time.Millisecond,
			waitForRotation:   true,
		},
		{name: "should not rotate certificate if checkInterval is longer than rotation check interval",
			rotations:         1,
			rotationsExpected: 0,
			checkInterval:     1 * time.Hour,
			waitForRotation:   false,
		},
		{name: "should not rotate certificate if checkInterval is longer than rotation check interval with multiple rotations",
			rotations:         2,
			rotationsExpected: 0,
			checkInterval:     1 * time.Hour,
			waitForRotation:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert := tassert.New(t)

			mrc := tresorFake.NewFakeMRC()
			cm := tresorFake.NewFakeWithMRC(mrc, tc.checkInterval)

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
			firstCert, _ := cm.IssueCertificate(certificate.ForCommonName("testhook.ns.svc"))

			server := NewServer("testhook", "ns", 6000, cm, nil, onCertChange)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err := server.Run(ctx)
			assert.NoError(err)

			// wait for the first cert to be issued
			<-wait

			for i := 0; i < tc.rotations; i++ {
				mrc.NewCertEvent(fmt.Sprintf("newcert-%d", i), constants.MRCStateIssuingRollout)
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

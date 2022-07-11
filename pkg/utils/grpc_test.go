package utils

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	tresorFake "github.com/openservicemesh/osm/pkg/certificate/providers/tresor/fake"
)

func TestNewGrpc(t *testing.T) {
	assert := tassert.New(t)
	certManager := tresorFake.NewFake(nil, 1*time.Hour)
	adsCert, err := certManager.IssueCertificate("fake-ads", certificate.Internal)
	assert.NoError(err)

	certPem := adsCert.GetCertificateChain()
	keyPem := adsCert.GetPrivateKey()
	rootPem := adsCert.GetTrustedCAs()
	var emptyByteArray []byte

	type newGrpcTest struct {
		serverType    string
		port          int
		certPem       []byte
		expectedError bool
	}

	newGrpcTests := []newGrpcTest{
		{"abc", 123, emptyByteArray, true},
		{"ADS", 8081, emptyByteArray, true},
		{"ADS", 8080, certPem, false},
	}

	for _, gt := range newGrpcTests {
		resServer, resListener, err := NewGrpc(gt.serverType, gt.port, gt.certPem, keyPem, rootPem)
		if err != nil {
			assert.Nil(resServer)
			assert.Nil(resListener)
			assert.True(gt.expectedError)
		} else {
			assert.NotNil(resServer)
			assert.NotNil(resListener)
			assert.False(gt.expectedError)
		}
	}
}

func TestGrpcServe(t *testing.T) {
	assert := tassert.New(t)

	certManager := tresorFake.NewFake(nil, 1*time.Hour)
	adsCert, err := certManager.IssueCertificate("fake-ads", certificate.Internal)

	assert.NoError(err)

	serverType := "ADS"
	port := 9999
	grpcServer, lis, err := NewGrpc(serverType, port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetTrustedCAs())
	assert.Nil(err)

	ctx, cancel := context.WithCancel(context.Background())
	errorCh := make(chan interface{}, 1)
	go GrpcServe(ctx, grpcServer, lis, cancel, serverType, errorCh)
	defer cancel()
	time.Sleep(50 * time.Millisecond)

	assert.Len(errorCh, 0)
}

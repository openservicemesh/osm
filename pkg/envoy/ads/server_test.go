package ads

import (
	"context"
	"testing"
	"time"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
)

func TestNewGrpc(t *testing.T) {
	assert := tassert.New(t)
	certManager := tresor.NewFakeCertManager(nil)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	certPem := adsCert.GetCertificateChain()
	keyPem := adsCert.GetPrivateKey()
	rootPem := adsCert.GetIssuingCA()
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

	certManager := tresor.NewFakeCertManager(nil)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	serverType := "ADS"
	port := 9999
	grpcServer, lis, err := NewGrpc(serverType, port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	assert.Nil(err)

	ctx, cancel := context.WithCancel(context.Background())
	errorCh := make(chan interface{}, 1)
	go GrpcServe(ctx, grpcServer, lis, cancel, serverType, errorCh)
	defer cancel()
	time.Sleep(50 * time.Millisecond)

	assert.Len(errorCh, 0)
}

func TestSetupMutualTLS(t *testing.T) {
	assert := tassert.New(t)

	type setupMutualTLStest struct {
		certPem       []byte
		keyPem        []byte
		ca            []byte
		expectedError string
	}

	certManager := tresor.NewFakeCertManager(nil)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	serverType := "ADS"
	goodCertPem := adsCert.GetCertificateChain()
	goodKeyPem := adsCert.GetPrivateKey()
	goodCA := adsCert.GetIssuingCA()
	var emptyByteArray []byte

	setupMutualTLStests := []setupMutualTLStest{
		{emptyByteArray, goodKeyPem, goodCA, "[grpc][mTLS][ADS] Failed loading Certificate ([]) and Key "},
		{goodCertPem, goodKeyPem, emptyByteArray, "[grpc][mTLS][ADS] Failed to append client certs"},
		{goodCertPem, goodKeyPem, goodCA, ""},
	}

	for _, smt := range setupMutualTLStests {
		result, err := setupMutualTLS(true, serverType, smt.certPem, smt.keyPem, smt.ca)
		if err != nil {
			assert.Nil(result)
			assert.Contains(err.Error(), smt.expectedError)
		} else {
			assert.NotNil(result)
			assert.Empty(smt.expectedError)
		}
	}
}

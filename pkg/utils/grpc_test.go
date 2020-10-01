package utils

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
)

func TestNewGrpc(t *testing.T) {
	assert := assert.New(t)
	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	certPem := adsCert.GetCertificateChain()
	keyPem := adsCert.GetPrivateKey()
	rootPem := adsCert.GetIssuingCA()
	emptyByteArray := []byte{}

	type newGrpcTest struct {
		serverType    string
		port          int
		certPem       []byte
		expectedError string
	}

	newGrpcTests := []newGrpcTest{
		{"abc", 123, emptyByteArray, "listen tcp :123: bind: permission denied"},
		{"ADS", 8081, emptyByteArray, "[grpc][mTLS][ADS] Failed loading Certificate ([]) and Key"},
		{"ADS", 8080, certPem, ""},
	}

	for _, gt := range newGrpcTests {
		resServer, resListener, err := NewGrpc(gt.serverType, gt.port, gt.certPem, keyPem, rootPem)
		if err != nil {
			assert.Nil(resServer)
			assert.Nil(resListener)
			assert.Contains(err.Error(), gt.expectedError)
		} else {
			assert.NotNil(resServer)
			assert.NotNil(resListener)
			assert.Empty(gt.expectedError)
		}
	}
}

func TestGrpcServe(t *testing.T) {
	assert := assert.New(t)

	cache := make(map[certificate.CommonName]certificate.Certificater)
	certManager := tresor.NewFakeCertManager(&cache, 1*time.Hour)
	adsCert, err := certManager.GetRootCertificate()
	assert.Nil(err)

	serverType := "ADS"
	flags := pflag.NewFlagSet(`osm-controller`, pflag.ExitOnError)
	port := flags.Int("port", constants.OSMControllerPort, "Aggregated Discovery Service port number.")
	grpcServer, lis, err := NewGrpc(serverType, *port, adsCert.GetCertificateChain(), adsCert.GetPrivateKey(), adsCert.GetIssuingCA())
	assert.Nil(err)

	ctx, cancel := context.WithCancel(context.Background())
	errorCh := make(chan interface{}, 1)
	go GrpcServe(ctx, grpcServer, lis, cancel, serverType, errorCh)
	defer cancel()
	time.Sleep(50 * time.Millisecond)

	assert.Len(errorCh, 0)
}

package utils

import (
	"context"
	"testing"
	"time"

	"github.com/openservicemesh/osm/pkg/certificate"
	"github.com/openservicemesh/osm/pkg/certificate/providers/tresor"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
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
		serverType string
		port       int
		certPem    []byte
	}

	newGrpcTests := []newGrpcTest{
		{"abc", 123, emptyByteArray},
		{"ADS", 8081, emptyByteArray},
		{"ADS", 8080, certPem},
	}

	for _, gt := range newGrpcTests {
		resServer, resListener, err := NewGrpc(gt.serverType, gt.port, gt.certPem, keyPem, rootPem)
		if err != nil {
			assert.Nil(resServer)
			assert.Nil(resListener)
			assert.NotNil(err)
		} else {
			assert.NotNil(resServer)
			assert.NotNil(resListener)
			assert.Nil(err)
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

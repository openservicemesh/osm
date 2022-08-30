package ads

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
	certManager := tresorFake.NewFake(1 * time.Hour)

	type newGrpcTest struct {
		serverType string
		port       int
		cm         *certificate.Manager
	}

	newGrpcTests := []newGrpcTest{
		{"ADS", 8080, certManager},
	}

	for _, gt := range newGrpcTests {
		resServer, resListener, err := NewGrpc(gt.serverType, gt.port, "fake-ads", certManager)
		assert.NotNil(resServer)
		assert.NotNil(resListener)
		assert.NoError(err)
	}
}

func TestGrpcServe(t *testing.T) {
	assert := tassert.New(t)

	certManager := tresorFake.NewFake(1 * time.Hour)

	serverType := "ADS"
	certName := "fake-ads"
	port := 9999
	grpcServer, lis, err := NewGrpc(serverType, port, certName, certManager)
	assert.Nil(err)

	ctx, cancel := context.WithCancel(context.Background())
	errorCh := make(chan interface{}, 1)
	err = grpcServer.GrpcServe(ctx, cancel, lis, errorCh)
	assert.NoError(err)
	defer cancel()
	time.Sleep(50 * time.Millisecond)

	assert.Len(errorCh, 0)
}

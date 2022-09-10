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
		serverType    string
		port          int
		cm            *certificate.Manager
		expectedError bool
	}

	newGrpcTests := []newGrpcTest{
		{"abc", 123, nil, true},
		{"ADS", 8080, certManager, false},
	}

	for _, gt := range newGrpcTests {
		resServer, resListener, err := NewGrpc(gt.serverType, gt.port, "fake-ads", certManager)
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

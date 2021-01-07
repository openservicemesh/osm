package utils

import (
	"context"
	"net"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"

	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetIPFromContext(t *testing.T) {
	assert := tassert.New(t)

	ctxList := map[context.Context]net.Addr{
		context.Background(): nil,
		peer.NewContext(context.TODO(), &peer.Peer{Addr: tests.NewMockAddress("9.8.7.6")}): tests.NewMockAddress("9.8.7.6"),
	}

	for ctx, expectedRes := range ctxList {
		result := GetIPFromContext(ctx)

		assert.Equal(expectedRes, result)
	}
}

package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/peer"

	"github.com/openservicemesh/osm/pkg/tests"
)

func TestGetIPFromContext(t *testing.T) {
	assert := assert.New(t)

	ctxs := []context.Context{
		context.Background(),
		peer.NewContext(context.TODO(), &peer.Peer{Addr: tests.NewMockAddress("9.8.7.6")}),
	}

	for _, ctx := range ctxs {
		result := GetIPFromContext(ctx)

		if result == nil {
			assert.Nil(result)
		} else {
			pk, _ := peer.FromContext(ctx)
			assert.Equal(result, pk.Addr)
		}
	}
}

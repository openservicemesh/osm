package config

import (
	"testing"

	fakeConfigClient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	tassert "github.com/stretchr/testify/assert"
)

func TestNewConfigClient(t *testing.T) {
	assert := tassert.New(t)

	client, err := newConfigClient(fakeConfigClient.NewSimpleClientset(), nil, nil)
	assert.Nil(err)
	assert.NotNil(client)
}

package ads

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestLiveness(t *testing.T) {
	assert := tassert.New(t)
	assert.True((&Server{}).Liveness())
}

func TestReadiness(t *testing.T) {
	assert := tassert.New(t)
	assert.True((&Server{ready: true}).Readiness())
}

func TestServerGetID(t *testing.T) {
	assert := tassert.New(t)
	actual := (&Server{}).GetID()
	assert.Equal(ServerType, actual)
}

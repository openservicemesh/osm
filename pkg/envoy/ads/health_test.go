package ads

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestLiveness(t *testing.T) {
	assert := tassert.New(t)

	exp := true
	res := (&Server{}).Liveness()
	assert.Equal(res, exp)
}

func TestReadiness(t *testing.T) {
	assert := tassert.New(t)

	s := Server{
		ready: true,
	}
	exp := true
	res := s.Readiness()
	assert.Equal(res, exp)
}

func TestServerGetID(t *testing.T) {
	assert := tassert.New(t)

	expected := ServerType
	actual := (&Server{}).GetID()
	assert.Equal(actual, expected)
}

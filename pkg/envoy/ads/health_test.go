package ads

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestServerGetID(t *testing.T) {
	assert := tassert.New(t)

	expected := ServerType
	actual := (&Server{}).GetID()
	assert.Equal(actual, expected)
}

package envoy

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestTypeURIString(t *testing.T) {
	assert := tassert.New(t)

	expected := "test-content"
	actual := TypeURI(expected).String()
	assert.Equal(actual, expected)
}

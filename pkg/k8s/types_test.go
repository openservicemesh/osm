package k8s

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestEventTypeString(t *testing.T) {
	assert := tassert.New(t)

	expected := "test-content"
	actual := EventType(expected).String()
	assert.Equal(actual, expected)
}

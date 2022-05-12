package utils

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestGetLastChunkOfSlashed(t *testing.T) {
	assert := tassert.New(t)

	type getLastChunkOfSlashedTest struct {
		input             string
		expectedLastChunk string
	}

	getLastChunkOfSlashedTests := []getLastChunkOfSlashedTest{
		{"a/b/c", "c"},
		{"abc", "abc"},
	}

	for _, lcst := range getLastChunkOfSlashedTests {
		result := GetLastChunkOfSlashed(lcst.input)

		assert.Equal(result, lcst.expectedLastChunk)
	}
}

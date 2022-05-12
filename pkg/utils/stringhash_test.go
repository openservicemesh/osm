package utils

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestHashFromString(t *testing.T) {
	assert := tassert.New(t)

	stringHash, err := HashFromString("some string")
	assert.NoError(err)
	assert.NotZero(stringHash)

	emptyStringHash, err := HashFromString("")
	assert.NoError(err)
	assert.Equal(emptyStringHash, uint64(14695981039346656037))

	assert.NotEqual(stringHash, emptyStringHash)
}

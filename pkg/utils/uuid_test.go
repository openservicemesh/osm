package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUUIDStr(t *testing.T) {
	assert := assert.New(t)

	output := NewUUIDStr()
	assert.NotEmpty(output)
}

func TestIsValidUUID(t *testing.T) {
	assert := assert.New(t)

	listUUID := map[string]bool{
		uuid.New().String():          true,
		uuid.New().String() + "!xyz": false,
	}

	for uuid, expectedBool := range listUUID {
		result := IsValidUUID(uuid)

		assert.Equal(expectedBool, result)
	}
}

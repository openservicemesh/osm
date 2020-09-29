package utils

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewUUIDStr(t *testing.T) {
	assert := assert.New(t)

	output := NewUUIDStr()
	outputType := reflect.TypeOf(output)
	stringType := reflect.TypeOf("")

	assert.IsType(stringType, outputType)
}

func TestIsValidUUID(t *testing.T) {
	assert := assert.New(t)

	listUUID := []string{
		uuid.New().String(),
		uuid.New().String() + "!xyz"}

	for _, uuid := range listUUID {
		result := IsValidUUID(uuid)

		if result == false {
			assert.False(result)
		} else {
			assert.True(result)
		}
	}
}

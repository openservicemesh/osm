package errcode

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	assert := tassert.New(t)
	assert.Equal(ErrInvalidCLIArgument.String(), "E1000")
}

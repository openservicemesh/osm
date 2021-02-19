package endpoint

import (
	"net"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	assert := tassert.New(t)
	ept := Endpoint{
		IP:   net.ParseIP("9.9.9.9"),
		Port: 1234,
	}
	assert.Equal(ept.String(), "(ip=9.9.9.9, port=1234)")
}

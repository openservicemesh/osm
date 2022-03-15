package certificate

import (
	"testing"
	time "time"

	tassert "github.com/stretchr/testify/assert"
)

func TestShouldRotate(t *testing.T) {
	assert := tassert.New(t)
	cert := &Certificate{
		Expiration: time.Now().Add(-1 * time.Hour),
	}
	assert.True(cert.ShouldRotate())

	cert = &Certificate{
		Expiration: time.Now().Add(time.Hour),
	}
	assert.False(cert.ShouldRotate())
}

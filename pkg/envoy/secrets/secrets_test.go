package secrets

import (
	"fmt"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/identity"
)

func TestNameForIdentity(t *testing.T) {
	testCases := []struct {
		si       identity.ServiceIdentity
		expected string
	}{
		{
			si:       identity.ServiceIdentity("foo.bar"),
			expected: "service-cert:bar/foo",
		},
		{
			si:       identity.ServiceIdentity("foo.baz"),
			expected: "service-cert:baz/foo",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test case %d", i), func(t *testing.T) {
			assert := tassert.New(t)

			actual := NameForIdentity(tc.si)
			assert.Equal(tc.expected, actual)
		})
	}
}

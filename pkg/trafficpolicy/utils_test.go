package trafficpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPRouteEqual(t *testing.T) {
	assert := assert.New(t)

	routes := []HTTPRoute{
		{
			PathRegex: "path",
			Methods:   []string{"GET"},
			Headers: map[string]string{
				"user-agent": "header1",
			},
		},
		{
			PathRegex: "path2",
			Methods:   []string{"GET"},
			Headers: map[string]string{
				"user-agent": "header2",
			},
		},
	}

	type httpRouteEqualTest struct {
		route1 HTTPRoute
		route2 HTTPRoute
		output bool
	}

	httpRouteEqualTests := []httpRouteEqualTest{
		{
			route1: routes[0],
			route2: routes[0],
			output: true,
		},
		{
			route1: routes[0],
			route2: routes[1],
			output: false,
		},
	}

	for _, test := range httpRouteEqualTests {
		equal := test.route1.Equal(test.route2)
		assert.Equal(equal, test.output)
	}
}

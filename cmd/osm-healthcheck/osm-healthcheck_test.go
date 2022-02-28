package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/constants"
)

func TestGetHealthcheckHander(t *testing.T) {
	testCases := []struct {
		name                 string
		hasOriginalTCPHeader bool
		expectedStatusCode   int
		shouldListen         bool
		expectedConnection   bool
	}{
		{
			name:                 "Bad request response when Original-Tcp-Port header is missing from request",
			hasOriginalTCPHeader: false,
			expectedStatusCode:   400,
			shouldListen:         false,
			expectedConnection:   false,
		},
		{
			name:                 "OK response",
			hasOriginalTCPHeader: true,
			expectedStatusCode:   200,
			shouldListen:         true,
			expectedConnection:   true,
		},
		{
			name:                 "Not found response when unable to establish connection",
			hasOriginalTCPHeader: true,
			expectedStatusCode:   404,
			shouldListen:         false,
			expectedConnection:   false,
		},
	}

	assert := tassert.New(t)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/osm-healthcheck", nil)
			if test.hasOriginalTCPHeader {
				req.Header.Add("Original-Tcp-Port", "3456")
			}

			if test.shouldListen {
				listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", constants.LocalhostIPAddress, "3456"))
				assert.Nil(err)
				//nolint: errcheck
				//#nosec G307
				defer listener.Close()

				go func() {
					conn, err := listener.Accept()
					assert.Nil(err)
					assert.Equal(test.expectedConnection, conn != nil)
					if conn != nil {
						err = conn.Close()
						assert.Nil(err)
					}
				}()
			}

			w := httptest.NewRecorder()

			healthcheckHandler(w, req)

			res := w.Result()
			assert.Equal(test.expectedStatusCode, res.StatusCode)
		})
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tassert "github.com/stretchr/testify/assert"
)

func TestGetHealthcheckHander(t *testing.T) {
	testCases := []struct {
		name                 string
		hasOriginalTCPHeader bool
		expectedStatusCode   int
	}{
		{
			name:                 "Bad request response when Original-Tcp-Port header is missing from request",
			hasOriginalTCPHeader: false,
			expectedStatusCode:   400,
		},
	}

	assert := tassert.New(t)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/osm-healthcheck", nil)
			if test.hasOriginalTCPHeader {
				req.Header.Add("Original-Tcp-Port", "3456")
			}
			w := httptest.NewRecorder()

			healthcheckHandler(w, req)

			res := w.Result()
			assert.Equal(test.expectedStatusCode, res.StatusCode)
		})
	}
}

package httpserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/openservicemesh/osm/pkg/debugger"
)

const (
	validRoutePath = "/debug/test1"
	responseBody   = "OSM rules"
)

func TestNewDebugHTTPServer(t *testing.T) {
	assert := assert.New(t)

	mockCtrl := gomock.NewController(t)
	mockDebugServer := debugger.NewMockDebugServer(mockCtrl)
	mockDebugServer.EXPECT().GetHandlers().Return(map[string]http.Handler{
		validRoutePath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = fmt.Fprint(w, responseBody)
		}),
	})
	debugServ := NewDebugHTTPServer(mockDebugServer, testPort).(*DebugServer)
	testDebug := &httptest.Server{
		Config: debugServ.Server,
	}

	type newDebugHTTPServerTest struct {
		routePath              string
		expectedHTTPStatusCode int
		expectedResponseBody   string
	}

	newDebugHTTPServerTests := []newDebugHTTPServerTest{
		{invalidRoutePath, http.StatusNotFound, "404 page not found\n"},
		{validRoutePath, http.StatusOK, responseBody},
	}

	for _, debugTest := range newDebugHTTPServerTests {
		req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, debugTest.routePath), nil)
		w := httptest.NewRecorder()
		testDebug.Config.Handler.ServeHTTP(w, req)
		resp := w.Result()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)

		assert.Equal(debugTest.expectedHTTPStatusCode, resp.StatusCode)
		assert.Equal(debugTest.expectedResponseBody, string(bodyBytes))
	}
}

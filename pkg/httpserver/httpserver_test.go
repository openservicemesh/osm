package httpserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openservicemesh/osm/pkg/debugger"
	"github.com/openservicemesh/osm/pkg/health"
	"github.com/openservicemesh/osm/pkg/metricsstore"
)

const (
	url              = "http://localhost"
	validRoutePath   = "/debug/test1"
	invalidRoutePath = "/debug/test2"
	testPort         = 9999
	responseBody     = "OSM rules"

	// These paths are internal to debugServer
	readyPath   = "/health/ready"
	alivePath   = "/health/alive"
	metricsPath = "/metrics"
)

// Dynamic variables for extended testing
var (
	readyResult      bool
	aliveResult      bool
	boolToRESTMapper = map[bool]int{
		true:  http.StatusOK,
		false: http.StatusServiceUnavailable,
	}
)

// For Probes, verifies and expects StatusCode to match mapped expectedResult
func checkResult(ts *httptest.Server, path string, expectedResult bool) {
	req := httptest.NewRequest("GET", path, nil)

	w := httptest.NewRecorder()
	ts.Config.Handler.ServeHTTP(w, req)
	resp := w.Result()

	Expect(resp.StatusCode).To(Equal(boolToRESTMapper[expectedResult]))
}

var _ = Describe("Test httpserver", func() {
	Context("HTTP OSM debug server", func() {
		metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")

		// Fake probes
		testProbe := health.FakeProbe{
			LivenessRet: func() bool {
				return aliveResult
			},
			ReadinessRet: func() bool {
				return readyResult
			},
		}

		// Fake debug server
		var fakeDebugServer debugger.FakeDebugServer = debugger.FakeDebugServer{
			Mappings: map[string]http.Handler{
				validRoutePath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = fmt.Fprintf(w, responseBody)
				}),
			},
		}

		httpServ := NewHTTPServer(testProbe, metricsStore, testPort, fakeDebugServer)

		testServer := httptest.Server{
			Config: httpServ.server,
		}

		It("should return 404 for a non-existant debug url", func() {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, invalidRoutePath), nil)

			w := httptest.NewRecorder()
			testServer.Config.Handler.ServeHTTP(w, req)

			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should return 200 for an existing debug url - body should match", func() {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, validRoutePath), nil)

			w := httptest.NewRecorder()
			testServer.Config.Handler.ServeHTTP(w, req)

			resp := w.Result()
			bodyBytes, _ := ioutil.ReadAll(resp.Body)

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(string(bodyBytes)).To(Equal(responseBody))
		})

		It("should hit proper liveness results", func() {
			checkResult(&testServer, fmt.Sprintf("%s%s", url, alivePath), aliveResult)
			// Swap query/expected result and test again
			aliveResult = !aliveResult
			checkResult(&testServer, fmt.Sprintf("%s%s", url, alivePath), aliveResult)
		})

		It("should hit proper readiness results", func() {
			checkResult(&testServer, fmt.Sprintf("%s%s", url, readyPath), readyResult)
			// Swap query/expected result and test again
			readyResult = !readyResult
			checkResult(&testServer, fmt.Sprintf("%s%s", url, readyPath), readyResult)
		})

		It("Should hit and read metrics path", func() {
			req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, metricsPath), nil)

			w := httptest.NewRecorder()
			testServer.Config.Handler.ServeHTTP(w, req)

			resp := w.Result()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})

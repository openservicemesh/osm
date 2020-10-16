package httpserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/golang/mock/gomock"
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

// Records an HTTP request and returns a response
func recordCall(ts *httptest.Server, path string) *http.Response {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	ts.Config.Handler.ServeHTTP(w, req)

	return w.Result()
}

var _ = Describe("Test httpserver", func() {
	var (
		mockCtrl        *gomock.Controller
		mockProbe       *health.MockProbes
		testServer      *httptest.Server
		mockDebugServer *debugger.MockDebugServer
	)
	mockCtrl = gomock.NewController(GinkgoT())

	BeforeEach(func() {
		mockProbe = health.NewMockProbes(mockCtrl)
		testProbes := []health.Probes{mockProbe}
		metricsStore := metricsstore.NewMetricStore("TBD_NameSpace", "TBD_PodName")

		mockDebugServer = debugger.NewMockDebugServer(mockCtrl)
		mockDebugServer.EXPECT().GetHandlers().Return(map[string]http.Handler{
			validRoutePath: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprint(w, responseBody)
			}),
		})

		httpServ := NewHTTPServer(testProbes, nil, metricsStore, testPort, mockDebugServer)
		testServer = &httptest.Server{
			Config: httpServ.server,
		}
	})

	It("should return 404 for a non-existent debug url", func() {
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

	It("should result in a successful readiness probe", func() {
		mockProbe.EXPECT().Readiness().Return(true).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, readyPath))

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("should result in an unsuccessful readiness probe", func() {
		mockProbe.EXPECT().Readiness().Return(false).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, readyPath))

		Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})

	It("should result in a successful liveness probe", func() {
		mockProbe.EXPECT().Liveness().Return(true).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, alivePath))

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("should result in an unsuccessful liveness probe", func() {
		mockProbe.EXPECT().Liveness().Return(false).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, alivePath))

		Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})

	It("should result in an unsuccessful probe when the probe path is incorrect", func() {
		mockProbe.EXPECT().Liveness().Times(0)
		mockProbe.EXPECT().Liveness().Times(0)
		mockProbe.EXPECT().GetID().Times(0)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, invalidRoutePath))

		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
	})

	It("Should hit and read metrics path", func() {
		req := httptest.NewRequest("GET", fmt.Sprintf("%s%s", url, metricsPath), nil)

		w := httptest.NewRecorder()
		testServer.Config.Handler.ServeHTTP(w, req)

		resp := w.Result()
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})

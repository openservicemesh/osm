package httpserver

import (
	"fmt"
	"io/ioutil"
	"net/http"

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
		httpServ.Start()

		It("should return 404 for a non-existant debug url", func() {
			resp, err := http.Get(fmt.Sprintf("%s:%d%s", url, testPort, invalidRoutePath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("should return 200 for an existing debug url - body should match", func() {
			resp, err := http.Get(fmt.Sprintf("%s:%d%s", url, testPort, validRoutePath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			bodyBytes, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()

			Expect(err).To(BeNil())
			Expect(string(bodyBytes)).To(Equal(responseBody))
		})

		It("should hit proper liveness results", func() {
			resp, err := http.Get(fmt.Sprintf("%s:%d%s", url, testPort, alivePath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(boolToRESTMapper[aliveResult]))

			// Swap result
			aliveResult = !aliveResult

			resp, err = http.Get(fmt.Sprintf("%s:%d%s", url, testPort, alivePath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(boolToRESTMapper[aliveResult]))
		})

		It("should hit proper readiness results", func() {
			resp, err := http.Get(fmt.Sprintf("%s:%d%s", url, testPort, readyPath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(boolToRESTMapper[readyResult]))

			// Swap result
			readyResult = !readyResult

			resp, err = http.Get(fmt.Sprintf("%s:%d%s", url, testPort, readyPath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(boolToRESTMapper[readyResult]))
		})

		It("Should hit and read metrics path", func() {
			resp, err := http.Get(fmt.Sprintf("%s:%d%s", url, testPort, metricsPath))

			Expect(err).To(BeNil())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("should stop with no errors", func() {
			err := httpServ.Stop()
			Expect(err).To(BeNil())
		})

	})
})

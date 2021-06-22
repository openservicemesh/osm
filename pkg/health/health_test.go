package health

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	url                = "http://localhost"
	testHTTPServerPort = 8888
	readyPath          = "/health/ready"
	alivePath          = "/health/alive"
)

// Records an HTTP request and returns a response
func recordCall(ts *httptest.Server, path string) *http.Response {
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()

	ts.Config.Handler.ServeHTTP(w, req)

	return w.Result()
}

var _ = Describe("test health probe helpers", func() {
	It("probes", func() {
		p := HTTPProbe{
			URL:      "http://localhost/a/b/c",
			Protocol: ProtocolHTTP,
		}
		actualResponseCode, err := p.Probe()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`refused`))
		Expect(actualResponseCode).To(Equal(503))
	})

	It("sets probe response", func() {
		w := httptest.NewRecorder()
		setProbeResponse(w, 987, "-the-message-")
		Expect(w.Body.String()).To(Equal("-the-message-"))
		Expect(w.Code).To(Equal(987))
	})
})

var _ = Describe("Test httpserver with probes", func() {
	var (
		mockCtrl   *gomock.Controller
		mockProbe  *MockProbes
		testServer *httptest.Server
	)
	mockCtrl = gomock.NewController(GinkgoT())

	BeforeEach(func() {
		mockProbe = NewMockProbes(mockCtrl)
		testProbes := []Probes{mockProbe}

		handlers := map[string]http.Handler{
			"/health/ready": ReadinessHandler(testProbes, nil),
			"/health/alive": LivenessHandler(testProbes, nil),
		}
		router := http.NewServeMux()
		for url, handler := range handlers {
			router.Handle(url, handler)
		}
		testServer = &httptest.Server{
			Config: &http.Server{
				Addr:    fmt.Sprintf(":%d", testHTTPServerPort),
				Handler: router,
			},
		}
	})

	It("should result in a successful readiness probe", func() {
		mockProbe.EXPECT().Readiness().Return(true).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, readyPath))

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("ignores this probe", func() {
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

	It("skips this probe", func() {
		mockProbe.EXPECT().Liveness().Return(false).Times(1)
		mockProbe.EXPECT().GetID().Return("test").Times(1)

		resp := recordCall(testServer, fmt.Sprintf("%s%s", url, alivePath))

		Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
	})
})

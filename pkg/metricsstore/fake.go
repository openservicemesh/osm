package metricsstore

import (
	"net/http"
	"time"
)

// NewFakeMetricStore return a fake metric store
func NewFakeMetricStore() MetricStore {
	return &fakeMetricStore{}
}

type fakeMetricStore struct{}

type fakeMetricHandler struct {
	metric string
}

func (m *fakeMetricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte(m.metric))
	if err != nil {
		log.Error().Err(err).Msgf("[%s] Error writing bytes", packageName)
	}
}

func (ms *fakeMetricStore) Start() {}

func (ms *fakeMetricStore) Stop() {}

func (ms *fakeMetricStore) Handler() http.Handler {
	return &fakeMetricHandler{metric: "OK"}
}

func (ms *fakeMetricStore) SetUpdateLatencySec(dur time.Duration) {}

func (ms *fakeMetricStore) IncArmAPIUpdateCallFailureCounter() {}

func (ms *fakeMetricStore) IncArmAPIUpdateCallSuccessCounter() {}

func (ms *fakeMetricStore) IncArmAPICallCounter() {}

func (ms *fakeMetricStore) IncK8sAPIEventCounter() {}

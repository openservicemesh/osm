package metricsstore

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMetricsStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Store Test Suite")
}

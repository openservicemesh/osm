package ads

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoyAds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy ADS Test Suite")
}

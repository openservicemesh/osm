package featureflags

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFeatureFlags(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Feature flags Test Suite")
}

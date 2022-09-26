package eds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoyEds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy EDS Test Suite")
}

package envoy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy Test Suite")
}

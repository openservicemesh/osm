package envoy

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoySds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy SDS Test Suite")
}

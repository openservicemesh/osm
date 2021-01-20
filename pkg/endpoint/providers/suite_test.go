package providers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEndpointProviders(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EndpointProviders Test Suite")
}

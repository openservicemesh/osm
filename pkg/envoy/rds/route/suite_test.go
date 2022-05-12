package route

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRouteConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Route Configuration Test Suite")
}

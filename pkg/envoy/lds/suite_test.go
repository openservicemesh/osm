package lds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoyLds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy LDS Test Suite")
}

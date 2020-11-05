package rds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoyRds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy RDS Test Suite")
}

package cds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvoyCds(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Envoy CDS Test Suite")
}

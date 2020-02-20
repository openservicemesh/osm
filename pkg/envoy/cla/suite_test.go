package cla

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClusterLoadAssignment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster Load Assignment Test Suite")
}

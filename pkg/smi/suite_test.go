package smi

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSMI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMI Test Suite")
}

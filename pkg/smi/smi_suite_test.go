package smi

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSMIModule(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMI Module Suite")
}

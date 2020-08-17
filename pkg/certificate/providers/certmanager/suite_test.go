package certmanager

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCertManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cert-manager Test Suite")
}

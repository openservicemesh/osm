package certificate

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCertificates(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Certificate Test Suite")
}

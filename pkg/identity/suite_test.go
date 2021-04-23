package identity

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHttpServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Identity Test Suite")
}

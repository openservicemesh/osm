package httpserver

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestHttpServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Http Server Test Suite")
}

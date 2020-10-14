package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Global context for now. This will prevent tests running parallelly though.
var td OsmTestData

// Since parseFlags is global, this is the Ginkgo way to do it.
// "init" is usually called by the go test runtime
// https://github.com/onsi/ginkgo/issues/265
func init() {
	registerFlags(&td)
}

// Cleanup when error
var _ = BeforeEach(func() {
	Expect(td.InitTestData(GinkgoT())).To(BeNil())
})

// Cleanup when error
var _ = AfterEach(func() {
	td.Cleanup(Test)
})

var _ = AfterSuite(func() {
	td.Cleanup(Suite)
})

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ginkgo e2e tests")
}

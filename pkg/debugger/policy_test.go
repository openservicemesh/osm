package providers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}


var _ = Describe("Test debugger method", func() {
	Context("Testing GetPolicy", func() {
		It("return policy", func() {
			ds:= debugServer{}
			actual:= ds.getPolicies()
			expected:= 123
			Expect(actual).To(Equal(expected))
		})
	})
})

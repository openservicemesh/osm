package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing utils helpers", func() {
	Context("Test GetLastNOfDotted", func() {
		It("Should return the last slice of a string split on a slash.", func() {
			Expect(GetLastNOfDotted("a.b.c", 2)).To(Equal("b.c"))
		})

		It("Should return the full string when there are no slashes.", func() {
			Expect(GetLastChunkOfSlashed("abc")).To(Equal("abc"))
		})
	})
})

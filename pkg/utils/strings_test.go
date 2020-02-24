package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing utils helpers", func() {
	Context("Test GetLastNOfDotted", func() {
		It("Should return the last slice of a string split on a dot.", func() {
			Expect(GetLastNOfDotted("a.b.c", 2)).To(Equal("b.c"))
		})

		It("Should return the full string when there are no dots.", func() {
			Expect(GetLastNOfDotted("abc", 0)).To(Equal("abc"))
		})
	})
})

var _ = Describe("Testing utils helpers", func() {
	Context("Test GetFirstOfDotted", func() {
		It("Should return the first slice of a string split on a dot.", func() {
			Expect(GetFirstOfDotted("a.b.c")).To(Equal("a"))
		})

		It("Should return the full string when there are no dots.", func() {
			Expect(GetFirstOfDotted("abc")).To(Equal("abc"))
		})
	})
})

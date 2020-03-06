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
			Expect(GetFirstNOfDotted("a.b.c.d", 3)).To(Equal("b/a"))
		})

		It("Should return the first slice of a string split on a dot.", func() {
			Expect(GetFirstNOfDotted("a.b.c", 2)).To(Equal("b/a"))
		})

		It("Should return the first slice of a string split on a dot.", func() {
			Expect(GetFirstNOfDotted("a.b.c.d.e", 4)).To(Equal("b/a"))
		})

		It("Should return the first slice of a string split on a dot.", func() {
			Expect(GetFirstNOfDotted("a.b", 1)).To(Equal("b/a"))
		})

		It("Should return the full string when there are no dots.", func() {
			Expect(GetFirstNOfDotted("abc", 0)).To(Equal("abc"))
		})
	})
})

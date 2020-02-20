package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing utils helpers", func() {
	Context("Test PrettyJSON", func() {
		It("should return pretty JSON and no error", func() {
			prettyJSON, err := PrettyJSON([]byte("{\"name\":\"baba yaga\"}"), "--prefix--")
			Expect(err).ToNot(HaveOccurred())
			Expect(prettyJSON).To(Equal([]byte(`{
--prefix--    "name": "baba yaga"
--prefix--}`)))
		})
	})

})

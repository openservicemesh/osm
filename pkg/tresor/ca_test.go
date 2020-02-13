package tresor

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test creation of a new CA", func() {
	Context("Create a new CA", func() {
		_, _, cert, _, err := NewCA("org", 2*time.Second)
		It("should create a new CA", func() {
			Expect(err).ToNot(HaveOccurred())
			Expect(cert.NotAfter.Sub(cert.NotBefore)).To(Equal(2 * time.Second))
			Expect(cert.Subject.Organization).To(Equal([]string{"org"}))
		})
	})
})

package service

import (
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test pkg/service functions", func() {
	defer GinkgoRecover()

	Context("Test ClusterName String method", func() {
		clusterNameStr := uuid.New().String()
		cn := ClusterName(clusterNameStr)

		It("implements stringer correctly", func() {
			Expect(cn.String()).To(Equal(clusterNameStr))
		})
	})
})

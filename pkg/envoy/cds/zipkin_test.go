package cds

import (
	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-service-mesh/osm/pkg/constants"
)

var _ = Describe("Test CDS Zipkin Configuration", func() {
	Context("Test getZipkinCluster()", func() {
		It("Returns Zipkin cluster config", func() {
			zipkinHostName := uuid.New().String()
			actual := getZipkinCluster(zipkinHostName)
			Expect(actual.Name).To(Equal(constants.EnvoyZipkinCluster))
			Expect(actual.AltStatName).To(Equal(constants.EnvoyZipkinCluster))
			Expect(len(actual.GetLoadAssignment().GetEndpoints())).To(Equal(1))
		})
	})
})

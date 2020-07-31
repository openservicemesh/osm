package cds

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Test CDS Zipkin Configuration", func() {
	Context("Test getZipkinCluster()", func() {
		It("Returns Zipkin cluster config", func() {
			cfg := configurator.NewFakeConfigurator()
			actual := getZipkinCluster(cfg)
			Expect(actual.Name).To(Equal(constants.EnvoyZipkinCluster))
			Expect(actual.AltStatName).To(Equal(constants.EnvoyZipkinCluster))
			Expect(len(actual.GetLoadAssignment().GetEndpoints())).To(Equal(1))
		})
	})
})

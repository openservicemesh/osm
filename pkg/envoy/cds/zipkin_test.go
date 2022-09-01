package cds

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Test CDS Tracing Configuration", func() {
	Context("Test getTracingCluster()", func() {
		It("Returns Tracing cluster config", func() {

			actual := *getTracingCluster(v1alpha2.MeshConfig{})
			Expect(actual.Name).To(Equal(constants.EnvoyTracingCluster))
			Expect(actual.AltStatName).To(Equal(constants.EnvoyTracingCluster))
			Expect(len(actual.GetLoadAssignment().GetEndpoints())).To(Equal(1))
		})
	})
})

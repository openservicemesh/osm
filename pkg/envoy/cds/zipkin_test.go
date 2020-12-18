package cds

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/constants"
)

var _ = Describe("Test CDS Tracing Configuration", func() {
	var (
		mockCtrl         *gomock.Controller
		mockConfigurator *configurator.MockConfigurator
	)

	mockCtrl = gomock.NewController(GinkgoT())
	mockConfigurator = configurator.NewMockConfigurator(mockCtrl)

	Context("Test getTracingCluster()", func() {
		It("Returns Tracing cluster config", func() {
			mockConfigurator.EXPECT().GetTracingHost().Return(constants.DefaultTracingHost).Times(1)
			mockConfigurator.EXPECT().GetTracingPort().Return(constants.DefaultTracingPort).Times(1)

			actual := *getTracingCluster(mockConfigurator)
			Expect(actual.Name).To(Equal(constants.EnvoyTracingCluster))
			Expect(actual.AltStatName).To(Equal(constants.EnvoyTracingCluster))
			Expect(len(actual.GetLoadAssignment().GetEndpoints())).To(Equal(1))
		})
	})
})

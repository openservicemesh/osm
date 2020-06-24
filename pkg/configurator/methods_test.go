package configurator

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test OSM controller plane Configurator module", func() {
	Context("create envoy config", func() {
		cfg := NewFakeConfigurator()

		It("returns the namespace in which OSM controller is installed", func() {
			actual := cfg.GetOSMNamespace()
			expected := "test-osm-namespace"
			Expect(actual).To(Equal(expected))
		})
	})
})

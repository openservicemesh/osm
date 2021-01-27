package featureflags

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test FeatureFlags", func() {
	Context("Testing OptionalFeatures", func() {
		It("should initialize OptionalFeatures", func() {
			defaultBackpressure := IsBackpressureEnabled()
			Expect(defaultBackpressure).ToNot(BeTrue())

			defaultRoutesV2 := IsRoutesV2Enabled()
			Expect(defaultRoutesV2).ToNot(BeTrue())

			optionalFeatures := OptionalFeatures{Backpressure: true, RoutesV2: true}
			Initialize(optionalFeatures)

			initializedBackpressure := IsBackpressureEnabled()
			Expect(initializedBackpressure).To(BeTrue())

			initializedRoutesV2 := IsRoutesV2Enabled()
			Expect(initializedRoutesV2).To(BeTrue())

		})

		It("should not re-initialize OptionalFeatures", func() {
			optionalFeatures2 := OptionalFeatures{Backpressure: false, RoutesV2: false}
			Initialize(optionalFeatures2)

			backpressure := IsBackpressureEnabled()
			Expect(backpressure).To(BeTrue())

			routesV2 := IsRoutesV2Enabled()
			Expect(routesV2).To(BeTrue())
		})
	})
})

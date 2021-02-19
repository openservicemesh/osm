package featureflags

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test FeatureFlags", func() {
	Context("Testing OptionalFeatures", func() {
		It("should initialize OptionalFeatures", func() {
			defaultRoutesV2 := IsRoutesV2Enabled()
			Expect(defaultRoutesV2).ToNot(BeTrue())

			defaultWASMStats := IsWASMStatsEnabled()
			Expect(defaultWASMStats).ToNot(BeTrue())

			optionalFeatures := OptionalFeatures{RoutesV2: true, WASMStats: true}
			Initialize(optionalFeatures)

			initializedRoutesV2 := IsRoutesV2Enabled()
			Expect(initializedRoutesV2).To(BeTrue())

			initializedWASMStats := IsWASMStatsEnabled()
			Expect(initializedWASMStats).To(BeTrue())

		})

		It("should not re-initialize OptionalFeatures", func() {
			optionalFeatures2 := OptionalFeatures{RoutesV2: false, WASMStats: false}
			Initialize(optionalFeatures2)

			routesV2 := IsRoutesV2Enabled()
			Expect(routesV2).To(BeTrue())

			WASMStats := IsWASMStatsEnabled()
			Expect(WASMStats).To(BeTrue())
		})
	})
})

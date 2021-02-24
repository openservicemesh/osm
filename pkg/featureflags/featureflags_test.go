package featureflags

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test FeatureFlags", func() {
	Context("Testing OptionalFeatures", func() {
		It("should initialize OptionalFeatures", func() {

			defaultWASMStats := IsWASMStatsEnabled()
			Expect(defaultWASMStats).ToNot(BeTrue())

			optionalFeatures := OptionalFeatures{WASMStats: true}
			Initialize(optionalFeatures)

			initializedWASMStats := IsWASMStatsEnabled()
			Expect(initializedWASMStats).To(BeTrue())

		})

		It("should not re-initialize OptionalFeatures", func() {
			optionalFeatures2 := OptionalFeatures{WASMStats: false}
			Initialize(optionalFeatures2)

			WASMStats := IsWASMStatsEnabled()
			Expect(WASMStats).To(BeTrue())
		})
	})
})

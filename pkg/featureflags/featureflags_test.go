package featureflags

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test FeatureFlags", func() {
	Context("Testing OptionalFeatures", func() {
		It("should initialize OptionalFeatures", func() {
			optionalFeatures := OptionalFeatures{Backpressure: true}
			Initialize(optionalFeatures)
			initializedBackpressure := IsBackpressureEnabled()
			Expect(initializedBackpressure).To(BeTrue())
		})
	})
})

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

			optionalFeatures := OptionalFeatures{Backpressure: true}
			Initialize(optionalFeatures)

			initializedBackpressure := IsBackpressureEnabled()
			Expect(initializedBackpressure).To(BeTrue())
		})

		It("should not re-initialize OptionalFeatures", func() {
			optionalFeatures2 := OptionalFeatures{Backpressure: false}
			Initialize(optionalFeatures2)

			backpressure := IsBackpressureEnabled()
			Expect(backpressure).To(BeTrue())
		})
	})
})

package injector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test Envoy configuration creation", func() {
	Context("create envoy config", func() {
		It("creates envoy config", func() {
			actual := getEnvoyConfigYAML()
			expected := envoyBootstrapConfigTmpl[1:]
			Expect(actual).To(Equal(expected))
		})
	})
})

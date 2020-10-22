package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test osm control plane installation with Helm", func() {
	Context("Using default values", func() {
		It("installs osm control plane successfully", func() {
			// Install OSM with Helm
			Expect(td.HelmInstallOSM()).To(Succeed())

		})
	})
})

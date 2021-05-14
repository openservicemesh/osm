package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test init-osm-controller functionalities",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 5,
	},
	func() {
		Context("When osm-controller starts in fresh environment", func() {
			It("creates default MeshConfig resource", func() {
				instOpts := Td.GetOSMInstallOpts()

				// Install OSM
				Expect(Td.InstallOSM(instOpts)).To(Succeed())
				meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate osm MeshConfig
				Expect(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.EnableEgress).Should(BeFalse())
				Expect(meshConfig.Spec.Sidecar.LogLevel).Should(Equal("debug"))
				Expect(meshConfig.Spec.Observability.PrometheusScraping).Should(BeTrue())
				Expect(meshConfig.Spec.Observability.EnableDebugServer).Should(BeTrue())
				Expect(meshConfig.Spec.Observability.Tracing.Enable).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.UseHTTPSIngress).Should(BeFalse())
				Expect(meshConfig.Spec.Certificate.ServiceCertValidityDuration).Should(Equal("24h"))
			})
		})
	})

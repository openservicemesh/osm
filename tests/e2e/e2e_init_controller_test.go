package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test osm-mesh-config functionalities",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 5,
	},
	func() {
		Context("When OSM is Installed", func() {
			It("create default MeshConfig resource", func() {

				if Td.InstType == "NoInstall" {
					Skip("Skipping test: NoInstall marked on a test that requires fresh installation")
				}
				instOpts := Td.GetOSMInstallOpts()

				// Install OSM
				Expect(Td.InstallOSM(instOpts)).To(Succeed())
				meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate osm MeshConfig
				Expect(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.EnableEgress).Should(BeFalse())
				Expect(meshConfig.Spec.Sidecar.LogLevel).Should(Equal("debug"))
				Expect(meshConfig.Spec.Observability.EnableDebugServer).Should(BeTrue())
				Expect(meshConfig.Spec.Observability.Tracing.Enable).Should(BeFalse())
				Expect(meshConfig.Spec.Certificate.ServiceCertValidityDuration).Should(Equal("24h"))
			})
		})
	})

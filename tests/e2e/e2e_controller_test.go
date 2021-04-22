package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

const meshConfigName = "osm-mesh-config"

var _ = OSMDescribe("Test osm-controller bootstrap initialization process",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("When osm-controller starts in fresh environment", func() {
			It("creates default MeshConfig resource", func() {
				instOpts := Td.GetOSMInstallOpts()
				namespace := instOpts.ControlPlaneNS

				// Install OSM
				Expect(Td.InstallOSM(instOpts)).To(Succeed())
				meshConfig, err := Td.GetMeshConfig(meshConfigName, namespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate osm MeshConfig
				Expect(meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.EnableEgress).Should(BeFalse())
				Expect(meshConfig.Spec.Sidecar.LogLevel).Should(Equal("error"))
				Expect(meshConfig.Spec.Observability.PrometheusScraping).Should(BeTrue())
				Expect(meshConfig.Spec.Observability.EnableDebugServer).Should(BeFalse())
				Expect(meshConfig.Spec.Observability.Tracing.Enable).Should(BeFalse())
				Expect(meshConfig.Spec.Traffic.UseHTTPSIngress).Should(BeFalse())
				Expect(meshConfig.Spec.Certificate.ServiceCertValidityDuration).Should(Equal("24h"))
			})
		})
	})

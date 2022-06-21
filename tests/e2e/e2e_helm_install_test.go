package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test osm control plane installation with Helm",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 4,
	},
	func() {
		Context("Helm install using default values", func() {
			It("installs osm control plane successfully", func() {
				if Td.InstType == NoInstall {
					Skip("Test is not going through InstallOSM, hence cannot be automatically skipped with NoInstall (#1908)")
				}

				namespace := "helm-install-namespace"
				release := "helm-install-osm"

				// Install OSM with Helm
				Expect(Td.HelmInstallOSM(release, namespace)).To(Succeed())

				meshConfig, err := Td.GetMeshConfig(namespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate osm MeshConfig
				spec := meshConfig.Spec
				Expect(spec.Traffic.EnablePermissiveTrafficPolicyMode).To(BeTrue())
				Expect(spec.Traffic.EnableEgress).To(BeTrue())
				Expect(spec.Sidecar.LogLevel).To(Equal("error"))
				Expect(spec.Observability.EnableDebugServer).To(BeFalse())
				Expect(spec.Observability.Tracing.Enable).To(BeFalse())
				Expect(spec.Certificate.ServiceCertValidityDuration).To(Equal("24h"))

				Expect(Td.DeleteHelmRelease(release, namespace)).To(Succeed())
			})
		})
	})

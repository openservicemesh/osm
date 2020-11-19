package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test osm control plane installation with Helm",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("Using default values", func() {
			It("installs osm control plane successfully", func() {
				if Td.InstType == NoInstall {
					Skip("Test is not going through InstallOSM, hence cannot be automatically skipped with NoInstall (#1908)")
				}

				namespace := "helm-install-namespace"
				release := "helm-install-osm"

				// Install OSM with Helm
				Expect(Td.HelmInstallOSM(release, namespace)).To(Succeed())

				configmap, err := Td.GetConfigMap("osm-config", namespace)
				Expect(err).ShouldNot(HaveOccurred())

				// validate osm configmap
				Expect(configmap.Data["permissive_traffic_policy_mode"]).Should(Equal("false"))
				Expect(configmap.Data["egress"]).Should(Equal("false"))
				Expect(configmap.Data["envoy_log_level"]).Should(Equal("error"))
				Expect(configmap.Data["enable_debug_server"]).Should(Equal("false"))
				Expect(configmap.Data["prometheus_scraping"]).Should(Equal("true"))
				Expect(configmap.Data["tracing_enable"]).Should(Equal("true"))
				Expect(configmap.Data["tracing_address"]).Should(Equal("jaeger.osm-system.svc.cluster.local"))
				Expect(configmap.Data["tracing_port"]).Should(Equal("9411"))
				Expect(configmap.Data["tracing_endpoint"]).Should(Equal("/api/v2/spans"))
				Expect(configmap.Data["use_https_ingress"]).Should(Equal("false"))
				Expect(configmap.Data["service_cert_validity_duration"]).Should(Equal("24h"))

				Expect(Td.DeleteHelmRelease(release, namespace)).To(Succeed())
			})
		})
	})

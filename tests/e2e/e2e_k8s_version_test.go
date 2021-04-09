package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

// This test is meant to install k8s clusters of different versions and run
// a correctness test to ensure OSM works on these versions.
var _ = OSMDescribe("Test HTTP traffic for different k8s versions",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("Version v1.19.7", func() {
			testK8sVersion("v1.19.7")
		})
		Context("Version v1.18.0", func() {
			testK8sVersion("v1.18.0")
		})
	})

func testK8sVersion(version string) {
	const sourceName = "client"
	const destName = "server"
	var ns = []string{sourceName, destName}

	Td.ClusterVersion = version // set the cluster version to test

	It("Tests HTTP traffic for client pod -> server pod", func() {
		if Td.InstType != KindCluster {
			Skip("Test is only meant to be run when installing a Kind cluster")
		}

		// Install OSM
		Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

		// Create Test NS
		for _, n := range ns {
			Expect(Td.CreateNs(n, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, n)).To(Succeed())
		}

		// Get simple pod definitions for the HTTP server
		svcAccDef, podDef, svcDef := Td.SimplePodApp(
			SimplePodAppDef{
				Name:      destName,
				Namespace: destName,
				Image:     "kennethreitz/httpbin",
				Ports:     []int{80},
			})

		_, err := Td.CreateServiceAccount(destName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destName, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1)).To(Succeed())

		srcPod := setupSource(sourceName, false /* no service for client */)

		By("Creating SMI policies")
		// Deploy allow rule client->server
		httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
			SimpleAllowPolicy{
				RouteGroupName:    "routes",
				TrafficTargetName: "test-target",

				SourceNamespace:      sourceName,
				SourceSVCAccountName: sourceName,

				DestinationNamespace:      destName,
				DestinationSvcAccountName: destName,
			})

		// Configs have to be put into a monitored NS
		_, err = Td.CreateHTTPRouteGroup(sourceName, httpRG)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateTrafficTarget(sourceName, trafficTarget)
		Expect(err).NotTo(HaveOccurred())

		// All ready. Expect client to reach server
		clientToServer := HTTPRequestDef{
			SourceNs:        sourceName,
			SourcePod:       srcPod.Name,
			SourceContainer: sourceName,

			Destination: fmt.Sprintf("%s.%s", dstSvc.Name, dstSvc.Namespace),
		}

		srcToDestStr := fmt.Sprintf("%s -> %s",
			fmt.Sprintf("%s/%s", sourceName, srcPod.Name),
			clientToServer.Destination)

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)

			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) HTTP Req failed %d %v",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, 90*time.Second)

		Expect(cond).To(BeTrue(), "Failed testing HTTP traffic for %s", srcToDestStr)
	})
}

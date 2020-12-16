package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test HTTP traffic from 1 pod client -> 1 pod server before and after osm-controller restart",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("SimpleClientServer traffic test involving osm-controller restart: HTTP", func() {
			testHTTPTrafficWithControllerRestart()
		})
	})

func testHTTPTrafficWithControllerRestart() {
	{
		const sourceName = "client"
		const destName = "server"
		var ns = []string{sourceName, destName}

		It("Tests HTTP traffic for client pod -> server pod", func() {
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

			Expect(cond).To(BeTrue(), "Failed testing HTTP traffic from source pod %s to destination service %s", srcPod.Name, dstSvc.Name)

			// Restart osm-controller
			By("Restarting OSM controller")
			Expect(Td.RestartOSMController(Td.GetOSMInstallOpts())).To(Succeed())

			// Expect client to reach server
			cond = Td.WaitForRepeatedSuccess(func() bool {
				result := Td.HTTPRequest(clientToServer)

				if result.Err != nil || result.StatusCode != 200 {
					Td.T.Logf("> (%s) HTTP Req failed %d %v",
						srcToDestStr, result.StatusCode, result.Err)
					return false
				}
				Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
				return true
			}, 20, 40*time.Second)
			Expect(cond).To(BeTrue(), "Failed testing HTTP traffic from source pod %s to destination service %s after osm-controller restart", srcPod.Name, dstSvc.Name)
		})
	}
}

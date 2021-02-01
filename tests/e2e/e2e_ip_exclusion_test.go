package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Tests traffic via IP range exclusion",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("Test IP range exclusion", func() {
			testIPExclusion()
		})
	})

func testIPExclusion() {
	{
		const sourceName = "client"
		const destName = "server"
		var ns = []string{sourceName, destName}

		It("Tests HTTP traffic to external server via IP exclusion", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			installOpts.EnablePermissiveMode = false // explicitly set to false to demonstrate IP exclusion
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			// Create Test NS
			for _, n := range ns {
				Expect(Td.CreateNs(n, nil)).To(Succeed())
			}
			// Only add source namespace to the mesh, destination is simulating an external cluster
			Expect(Td.AddNsToMesh(true, sourceName)).To(Succeed())

			// Set up the destination HTTP server. It is not part of the mesh
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

			// The destination IP will be programmed as an IP exclusion
			destinationIPRange := fmt.Sprintf("%s/32", dstSvc.Spec.ClusterIP)
			Expect(Td.UpdateOSMConfig("outbound_ip_range_exclusion_list", destinationIPRange))

			srcPod := setupSource(sourceName, false)

			By("Using IP range exclusion to access destination")
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

			Expect(cond).To(BeTrue(), "Failed testing HTTP traffic from source pod %s to destination %s", srcPod.Name, destinationIPRange)
		})
	}
}

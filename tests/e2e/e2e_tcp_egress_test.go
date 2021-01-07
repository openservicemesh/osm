package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test TCP traffic from 1 pod client -> egress server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 1,
	},
	func() {
		Context("SimpleClientServer egress TCP", func() {
			testTCPEgressTraffic()
		})
	})

func testTCPEgressTraffic() {
	{
		const sourceName = "client"
		const destName = "egress-server"

		It("Tests TCP traffic for client pod -> server pod", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			installOpts.EgressEnabled = true
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			// Load TCP server image
			Expect(Td.LoadImagesToKind([]string{"tcp-echo-server"})).To(Succeed())

			// Create client ns and add it to the mesh
			Expect(Td.CreateNs(sourceName, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, sourceName)).To(Succeed())

			// Create external ns for server not part of the mesh
			Expect(Td.CreateNs(destName, nil)).To(Succeed())

			destinationPort := 80

			// Get simple pod definitions for the TCP server
			svcAccDef, podDef, svcDef := Td.SimplePodApp(
				SimplePodAppDef{
					Name:        destName,
					Namespace:   destName,
					Image:       fmt.Sprintf("%s/tcp-echo-server:%s", installOpts.ContainerRegistryLoc, installOpts.OsmImagetag),
					Command:     []string{"/tcp-echo-server"},
					Args:        []string{"--port", fmt.Sprintf("%d", destinationPort)},
					Ports:       []int{destinationPort},
					AppProtocol: AppProtocolTCP,
				})

			_, err := Td.CreateServiceAccount(destName, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(destName, podDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(destName, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(destName, 120*time.Second, 1)).To(Succeed())

			srcPod := setupSource(sourceName, false /* no kubernetes service for the client */)

			// All ready. Expect client to reach server
			requestMsg := "test request"
			clientToServer := TCPRequestDef{
				SourceNs:        sourceName,
				SourcePod:       srcPod.Name,
				SourceContainer: sourceName,

				DestinationHost: fmt.Sprintf("%s.%s", dstSvc.Name, dstSvc.Namespace),
				DestinationPort: destinationPort,
				Message:         requestMsg,
			}

			srcToDestStr := fmt.Sprintf("%s -> %s:%d",
				fmt.Sprintf("%s/%s", sourceName, srcPod.Name),
				clientToServer.DestinationHost, clientToServer.DestinationPort)

			cond := Td.WaitForRepeatedSuccess(func() bool {
				result := Td.TCPRequest(clientToServer)

				if result.Err != nil {
					Td.T.Logf("> (%s) TCP Req failed, response: %s, err: %s",
						srcToDestStr, result.Response, result.Err)
					return false
				}

				// Ensure the echo response contains request message
				if !strings.Contains(result.Response, requestMsg) {
					Td.T.Logf("> (%s) Unexpected response: %s.\n Response expected to contain: %s", result.Response, requestMsg)
					return false
				}
				Td.T.Logf("> (%s) TCP Req succeeded, response: %s", srcToDestStr, result.Response)
				return true
			}, 5, 90*time.Second)

			Expect(cond).To(BeTrue(), "Failed testing TCP traffic from %s", srcToDestStr)

			By("Disabling egress")
			Expect(Td.UpdateOSMConfig("egress", "false")).To(Succeed())

			// Expect client not to reach server
			cond = Td.WaitForRepeatedSuccess(func() bool {
				result := Td.TCPRequest(clientToServer)

				if result.Err == nil {
					Td.T.Logf("> (%s) TCP Req did not fail, expected it to fail,  response: %s", srcToDestStr, result.Response)
					return false
				}
				Td.T.Logf("> (%s) TCP Req failed correctly, response: %s, err: %s", srcToDestStr, result.Response, result.Err)
				return true
			}, 5, 150*time.Second)
			Expect(cond).To(BeTrue())
		})
	}
}

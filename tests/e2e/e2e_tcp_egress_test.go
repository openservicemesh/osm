package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test TCP traffic from 1 pod client -> egress server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 9,
		// TODO(#1610): This test assumes that the user can create a pod that is not part of the mesh.
		// On Windows we set the HNS policies on all pods on the cluster and as a result we can't
		// have pods that are not part of the mesh. This will be resolved when OSM CNI is available.
		OS: constants.OSLinux,
	},
	func() {
		Context("SimpleClientServer egress TCP", func() {
			testTCPEgressTraffic()
		})
	})

func testTCPEgressTraffic() {
	const sourceName = "client"
	const destName = "egress-server"

	It("Tests TCP traffic for client pod -> server pod", func() {
		// Install OSM
		installOpts := Td.GetOSMInstallOpts()
		installOpts.EgressEnabled = true
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		meshConfig, _ := Td.GetMeshConfig(Td.OsmNamespace)

		// Create client ns and add it to the mesh
		Expect(Td.CreateNs(sourceName, nil)).To(Succeed())
		Expect(Td.AddNsToMesh(true, sourceName)).To(Succeed())

		// Create external ns for server not part of the mesh
		Expect(Td.CreateNs(destName, nil)).To(Succeed())

		destinationPort := fortioTCPPort

		// Get simple pod definitions for the TCP server
		svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
			SimplePodAppDef{
				PodName:     destName,
				Namespace:   destName,
				Image:       fortioImageName,
				Ports:       []int{destinationPort},
				AppProtocol: constants.ProtocolTCP,
				OS:          Td.ClusterOS,
			})
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destName, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destName, 120*time.Second, 1, nil)).To(Succeed())

		srcPod := setupFortioSource(sourceName, false /* no kubernetes service for the client */)

		// All ready. Expect client to reach server
		requestMsg := "test-request"
		clientToServer := TCPRequestDef{
			SourceNs:        sourceName,
			SourcePod:       srcPod.Name,
			SourceContainer: srcPod.Name,

			DestinationHost: fmt.Sprintf("%s.%s", dstSvc.Name, dstSvc.Namespace),
			DestinationPort: destinationPort,
			Message:         requestMsg,
		}

		srcToDestStr := fmt.Sprintf("%s -> %s:%d",
			fmt.Sprintf("%s/%s", sourceName, srcPod.Name),
			clientToServer.DestinationHost, clientToServer.DestinationPort)

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.FortioTCPLoadTest(FortioTCPLoadTestDef{TCPRequestDef: clientToServer, FortioLoadTestSpec: fortioSingleCallSpec})

			if result.Err != nil {
				Td.T.Logf("> (%s) TCP Req failed, return codes: %v, err: %s",
					srcToDestStr, result.AllReturnCodes(), result.Err)
				return false
			}

			// Ensure the correct TCP return code
			for retCode := range result.ReturnCodes {
				if retCode != fortioTCPRetCodeSuccess {
					Td.T.Logf("> (%s) Unexpected return code: %s.\n Return code expected to contain: %s. Skip.", srcToDestStr, retCode, fortioTCPRetCodeSuccess)
					continue
				}
				Td.T.Logf("> (%s) TCP Req succeeded, return code: %s", srcToDestStr, retCode)
				return true
			}
			return false
		}, 5, 90*time.Second)

		Expect(cond).To(BeTrue(), "Failed testing TCP traffic from %s", srcToDestStr)

		By("Disabling egress")
		meshConfig.Spec.Traffic.EnableEgress = false
		_, err = Td.UpdateOSMConfig(meshConfig)
		Expect(err).NotTo(HaveOccurred())

		// Expect client not to reach server
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.FortioTCPLoadTest(FortioTCPLoadTestDef{TCPRequestDef: clientToServer, FortioLoadTestSpec: fortioSingleCallSpec})

			// Look for failed TCP return code
			for retCode := range result.ReturnCodes {
				if retCode != fortioTCPRetCodeSuccess {
					Td.T.Logf("> (%s) TCP Req failed correctly, return codes: %s, err: %s", srcToDestStr, retCode, result.Err)
					return true
				}
			}

			Td.T.Logf("> (%s) TCP Req did not fail, expected it to fail, return codes: %v", srcToDestStr, result.AllReturnCodes())
			return false
		}, 5, 150*time.Second)
		Expect(cond).To(BeTrue())
	})
}

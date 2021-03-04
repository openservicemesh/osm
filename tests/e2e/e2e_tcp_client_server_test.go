package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test TCP traffic from 1 pod client -> 1 pod server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("SimpleClientServer TCP with SMI policies", func() {
			testTCPTraffic(false)
		})

		Context("SimpleClientServer TCP in permissive mode", func() {
			testTCPTraffic(true)
		})
	})

func testTCPTraffic(permissiveMode bool) {
	{
		const sourceName = "client"
		const destName = "server"
		var ns = []string{sourceName, destName}

		It("Tests TCP traffic for client pod -> server pod", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			installOpts.EnablePermissiveMode = permissiveMode
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			// Load TCP server image
			Expect(Td.LoadImagesToKind([]string{"tcp-echo-server"})).To(Succeed())

			// Create Test NS
			for _, n := range ns {
				Expect(Td.CreateNs(n, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, n)).To(Succeed())
			}

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

			trafficTargetName := "test-target"
			trafficRouteName := "routes"

			if !permissiveMode {
				By("Creating SMI policies")
				// Deploy allow rule client->server
				tcpRoute, trafficTarget := Td.CreateSimpleTCPAllowPolicy(
					SimpleAllowPolicy{
						RouteGroupName:    trafficRouteName,
						TrafficTargetName: trafficTargetName,

						SourceNamespace:      sourceName,
						SourceSVCAccountName: sourceName,

						DestinationNamespace:      destName,
						DestinationSvcAccountName: destName,
					},
					destinationPort,
				)

				// Configs have to be put into a monitored NS
				_, err = Td.CreateTCPRoute(sourceName, tcpRoute)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(sourceName, trafficTarget)
				Expect(err).NotTo(HaveOccurred())
			}

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
					Td.T.Logf("> (%s) Unexpected response: %s.\n Response expected to contain: %s", srcToDestStr, result.Response, requestMsg)
					return false
				}
				Td.T.Logf("> (%s) TCP Req succeeded, response: %s", srcToDestStr, result.Response)
				return true
			}, 5, 90*time.Second)

			Expect(cond).To(BeTrue(), "Failed testing TCP traffic from %s", srcToDestStr)

			if !permissiveMode {
				By("Deleting SMI policies")
				Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(sourceName).Delete(context.TODO(), trafficTargetName, metav1.DeleteOptions{})).To(Succeed())
				Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().TCPRoutes(sourceName).Delete(context.TODO(), trafficRouteName, metav1.DeleteOptions{})).To(Succeed())

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
			}
		})
	}
}

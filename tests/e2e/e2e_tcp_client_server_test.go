package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test TCP traffic from 1 pod client -> 1 pod server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 8,
		OS:     OSCrossPlatform,
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
	var (
		sourceNs = framework.RandomNameWithPrefix("client")
		destNs   = framework.RandomNameWithPrefix("server")
		ns       = []string{sourceNs, destNs}
	)

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
		svcAccDef, podDef, svcDef, err := Td.GetOSSpecificTCPEchoPod(framework.RandomNameWithPrefix("pod"), destNs, destinationPort)

		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destNs, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destNs, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destNs, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destNs, 120*time.Second, 1, nil)).To(Succeed())

		srcPod := setupSource(sourceNs, false /* no kubernetes service for the client */)

		trafficTargetName := "test-target"
		trafficRouteName := "routes"

		if !permissiveMode {
			By("Creating SMI policies")
			// Deploy allow rule client->server
			tcpRoute, trafficTarget := Td.CreateSimpleTCPAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    trafficRouteName,
					TrafficTargetName: trafficTargetName,

					SourceNamespace:      srcPod.Namespace,
					SourceSVCAccountName: srcPod.Spec.ServiceAccountName,

					DestinationNamespace:      destNs,
					DestinationSvcAccountName: svcAccDef.Name,
				},
				destinationPort,
			)

			// Configs have to be put into a monitored NS
			_, err = Td.CreateTCPRoute(destNs, tcpRoute)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(destNs, trafficTarget)
			Expect(err).NotTo(HaveOccurred())
		}

		// All ready. Expect client to reach server
		requestMsg := "test request"
		clientToServer := TCPRequestDef{
			SourceNs:        sourceNs,
			SourcePod:       srcPod.Name,
			SourceContainer: srcPod.Name,

			DestinationHost: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
			DestinationPort: destinationPort,
			Message:         requestMsg,
		}

		srcToDestStr := fmt.Sprintf("%s -> %s:%d",
			fmt.Sprintf("%s/%s", sourceNs, srcPod.Name),
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
		}, 5, Td.ReqSuccessTimeout)

		Expect(cond).To(BeTrue(), "Failed testing TCP traffic from %s", srcToDestStr)

		if !permissiveMode {
			By("Deleting SMI policies")
			Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(destNs).Delete(context.TODO(), trafficTargetName, metav1.DeleteOptions{})).To(Succeed())
			Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().TCPRoutes(destNs).Delete(context.TODO(), trafficRouteName, metav1.DeleteOptions{})).To(Succeed())

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

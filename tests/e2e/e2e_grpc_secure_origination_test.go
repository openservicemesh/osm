package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

const grpcbinSecurePort = 9001

// This test originates secure (over TLS) gRPC traffic between a client and server and
// verifies that the traffic is routable within the service mesh using TCP routes.
// <Client app with TLS origination> --> <local proxy> -- |mTLS over TCP routes| --> <remote proxy> -- <server app with TLS termination>
var _ = OSMDescribe("gRPC secure traffic origination for client pod -> server pod usint TCP routes",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 3,
	},
	func() {
		Context("gRPC secure traffic origination over HTTP2 with SMI TCP routes", func() {
			testSecureGRPCTraffic()
		})
	})

func testSecureGRPCTraffic() {
	{
		const sourceName = "client"
		const destName = "server"
		var ns = []string{sourceName, destName}

		It("Tests secure gRPC traffic origination for client pod -> server pod using TCP routes", func() {
			// Install OSM
			Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

			// Create Test NS
			for _, n := range ns {
				Expect(Td.CreateNs(n, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, n)).To(Succeed())
			}

			// Get simple pod definitions for the gRPC server
			svcAccDef, podDef, svcDef := Td.SimplePodApp(
				SimplePodAppDef{
					Name:        destName,
					Namespace:   destName,
					Image:       "moul/grpcbin",
					Ports:       []int{grpcbinSecurePort},
					AppProtocol: "tcp",
				})

			_, err := Td.CreateServiceAccount(destName, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(destName, podDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(destName, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1)).To(Succeed())

			srcPod := setupGRPCClient(sourceName)

			trafficTargetName := "test-target"
			trafficRouteName := "routes"

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
				}, grpcbinSecurePort)

			// Create SMI policies
			_, err = Td.CreateTCPRoute(sourceName, tcpRoute)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(sourceName, trafficTarget)
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
			clientToServer := GRPCRequestDef{
				SourceNs:        sourceName,
				SourcePod:       srcPod.Name,
				SourceContainer: sourceName,

				Destination: fmt.Sprintf("%s.%s:%d", dstSvc.Name, dstSvc.Namespace, grpcbinSecurePort),

				JSONRequest: "{\"greeting\": \"client\"}",
				Symbol:      "hello.HelloService/SayHello",
				UseTLS:      true, // Secure gRPC, service mesh will route this traffic using TCP routes over mTLS
			}

			srcToDestStr := fmt.Sprintf("%s/%s -> %s",
				clientToServer.SourceNs, clientToServer.SourcePod, clientToServer.Destination)

			cond := Td.WaitForRepeatedSuccess(func() bool {
				result := Td.GRPCRequest(clientToServer)

				if result.Err != nil {
					Td.T.Logf("> (%s) gRPC req failed, response: %s, err: %s",
						srcToDestStr, result.Response, result.Err)
					return false
				}

				Td.T.Logf("> (%s) gRPC req succeeded, response: %s", srcToDestStr, result.Response)
				return true
			}, 5, 3600*time.Second)

			Expect(cond).To(BeTrue(), "Failed testing gRPC traffic for: %s", srcToDestStr)

			By("Deleting SMI policies")
			Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(sourceName).Delete(context.TODO(), trafficTargetName, metav1.DeleteOptions{})).To(Succeed())
			Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().TCPRoutes(sourceName).Delete(context.TODO(), trafficRouteName, metav1.DeleteOptions{})).To(Succeed())

			// Expect client not to reach server
			cond = Td.WaitForRepeatedSuccess(func() bool {
				result := Td.GRPCRequest(clientToServer)

				if result.Err == nil {
					Td.T.Logf("> (%s) gRPC req did not fail, expected it to fail,  response: %s", srcToDestStr, result.Response)
					return false
				}
				Td.T.Logf("> (%s) gRPC req failed correctly, response: %s, err: %s", srcToDestStr, result.Response, result.Err)
				return true
			}, 5, 150*time.Second)
			Expect(cond).To(BeTrue())
		})
	}
}

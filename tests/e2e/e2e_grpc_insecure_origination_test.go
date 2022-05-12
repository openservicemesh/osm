package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
)

// This test originates insecure (plaintext) gRPC traffic between a client and server and
// verifies that the traffic is routable within the service mesh using HTTP routes.
// <Client app plaintext request> --> <local proxy> -- |mTLS over HTTP routes| --> <remote proxy> -- <server app plaintext request>
var _ = OSMDescribe("gRPC insecure traffic origination for client pod -> server pod usint HTTP routes",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 3,
	},
	func() {
		Context("gRPC insecure traffic origination over HTTP2 with SMI HTTP routes", func() {
			testGRPCTraffic()
		})
	})

func testGRPCTraffic() {
	var (
		sourceNs = framework.RandomNameWithPrefix("client")
		destNs   = framework.RandomNameWithPrefix("server")
		ns       = []string{sourceNs, destNs}
	)

	It("Tests insecure gRPC traffic origination for client pod -> server pod using HTTP routes", func() {
		// Install OSM
		Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

		// Create Test NS
		for _, n := range ns {
			Expect(Td.CreateNs(n, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, n)).To(Succeed())
		}

		// Get simple pod definitions for the gRPC server
		svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
			SimplePodAppDef{
				PodName:     framework.RandomNameWithPrefix("pod"),
				Namespace:   destNs,
				Image:       fortioImageName,
				Ports:       []int{fortioGRPCPort},
				AppProtocol: "grpc",
				OS:          Td.ClusterOS,
			})
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destNs, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destNs, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destNs, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destNs, 90*time.Second, 1, nil)).To(Succeed())

		srcPod := setupFortioClient(sourceNs)

		trafficTargetName := "test-target" //nolint: goconst
		trafficRouteName := "routes"       //nolint: goconst

		By("Creating SMI policies")
		// Deploy allow rule client->server
		httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
			SimpleAllowPolicy{
				RouteGroupName:    trafficRouteName,
				TrafficTargetName: trafficTargetName,

				SourceNamespace:      sourceNs,
				SourceSVCAccountName: srcPod.Spec.ServiceAccountName,

				DestinationNamespace:      destNs,
				DestinationSvcAccountName: svcAccDef.Name,
			})

		// Configs have to be put into a monitored NS
		_, err = Td.CreateHTTPRouteGroup(destNs, httpRG)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateTrafficTarget(destNs, trafficTarget)
		Expect(err).NotTo(HaveOccurred())

		// All ready. Expect client to reach server
		clientToServer := GRPCRequestDef{
			SourceNs:        sourceNs,
			SourcePod:       srcPod.Name,
			SourceContainer: srcPod.Name,

			Destination: fmt.Sprintf("%s.%s:%d", dstSvc.Name, dstSvc.Namespace, fortioGRPCPort),

			UseTLS: false, // insecure gRPC, service mesh will upgrade connection to mTLS
		}

		srcToDestStr := fmt.Sprintf("%s/%s -> %s",
			clientToServer.SourceNs, clientToServer.SourcePod, clientToServer.Destination)

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.FortioGRPCLoadTest(FortioGRPCLoadTestDef{GRPCRequestDef: clientToServer, FortioLoadTestSpec: fortioSingleCallSpec})

			// Ensure the correct GRPC return code
			for retCode := range result.ReturnCodes {
				if retCode != fortioGRPCRetCodeSuccess {
					Td.T.Logf("> (%s) Unexpected return code: %s.\n Return code expected to be: %s. ", srcToDestStr, retCode, fortioGRPCRetCodeSuccess)
					continue
				}
				Td.T.Logf("> (%s) gRPC Req succeeded, return code: %s", srcToDestStr, retCode)
				return true
			}
			return false
		}, 5, 90*time.Second)

		Expect(cond).To(BeTrue(), "Failed testing gRPC traffic for: %s", srcToDestStr)

		By("Deleting SMI policies")
		Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(destNs).Delete(context.TODO(), trafficTargetName, metav1.DeleteOptions{})).To(Succeed())
		Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(destNs).Delete(context.TODO(), trafficRouteName, metav1.DeleteOptions{})).To(Succeed())

		// Expect client not to reach server
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.FortioGRPCLoadTest(FortioGRPCLoadTestDef{GRPCRequestDef: clientToServer, FortioLoadTestSpec: fortioSingleCallSpec})

			// Look for failed GRPC return code
			for retCode := range result.ReturnCodes {
				if retCode != fortioGRPCRetCodeSuccess {
					Td.T.Logf("> (%s) gRPC Req failed correctly, return code: %s, err: %s", srcToDestStr, retCode, result.Err)
					return true
				}
			}

			Td.T.Logf("> (%s) gRPC Req did not fail, expected it to fail, return codes: %v", srcToDestStr, result.AllReturnCodes())
			return false
		}, 5, 150*time.Second)
		Expect(cond).To(BeTrue())
	})
}

func setupFortioClient(sourceName string) *corev1.Pod {
	// Get simple Pod definitions for the client
	svcAccDef, podDef, _, err := Td.SimplePodApp(SimplePodAppDef{
		PodName:   sourceName,
		Namespace: sourceName,
		Image:     fortioImageName,
		OS:        Td.ClusterOS,
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(sourceName, &svcAccDef)
	Expect(err).NotTo(HaveOccurred())

	srcPod, err := Td.CreatePod(sourceName, podDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(sourceName, 90*time.Second, 1, nil)).To(Succeed())

	return srcPod
}

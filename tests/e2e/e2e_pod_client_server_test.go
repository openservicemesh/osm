package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test HTTP traffic from 1 pod client -> 1 pod server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 7,
		OS:     OSCrossPlatform,
	},
	func() {
		Context("Test traffic flowing from client to server with a Kubernetes Service for the Source: HTTP", func() {
			withSourceKubernetesService := true
			testTraffic(withSourceKubernetesService)
		})

		Context("Test traffic flowing from client to server without a Kubernetes Service for the Source: HTTP", func() {
			// Prior iterations of OSM required that a source pod belong to a Kubernetes service
			// for the Envoy proxy to be configured for outbound traffic to some remote server.
			// This test ensures we test this scenario: client Pod is not associated w/ a service.
			withSourceKubernetesService := false
			testTraffic(withSourceKubernetesService)
		})
	})

func testTraffic(withSourceKubernetesService bool) {
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
		svcAccDef, podDef, svcDef, err := Td.GetOSSpecificHTTPBinPod(destName, destName)
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destName, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destName, podDef)
		Expect(err).NotTo(HaveOccurred())
		dstSvc, err := Td.CreateService(destName, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destName, 90*time.Second, 1, nil)).To(Succeed())

		srcPod := setupSource(sourceName, withSourceKubernetesService)

		By("Creating SMI policies")
		// Deploy allow rule client->server
		httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
			SimpleAllowPolicy{
				RouteGroupName:    "routes",
				TrafficTargetName: "test-target",

				SourceNamespace:      sourceName,
				SourceSVCAccountName: srcPod.Spec.ServiceAccountName,

				DestinationNamespace:      destName,
				DestinationSvcAccountName: svcAccDef.Name,
			})

		// Configs have to be put into a monitored NS
		_, err = Td.CreateHTTPRouteGroup(destName, httpRG)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateTrafficTarget(destName, trafficTarget)
		Expect(err).NotTo(HaveOccurred())

		// All ready. Expect client to reach server
		clientToServer := HTTPRequestDef{
			SourceNs:        sourceName,
			SourcePod:       srcPod.Name,
			SourceContainer: sourceName,

			Destination: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
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
		}, 5, Td.ReqSuccessTimeout)

		sourceService := map[bool]string{true: "with", false: "without"}[withSourceKubernetesService]
		Expect(cond).To(BeTrue(), "Failed testing HTTP traffic from source pod %s Kubernetes Service to a destination", sourceService)

		By("Deleting SMI policies")
		Expect(Td.SmiClients.AccessClient.AccessV1alpha3().TrafficTargets(destName).Delete(context.TODO(), trafficTarget.Name, metav1.DeleteOptions{})).To(Succeed())
		Expect(Td.SmiClients.SpecClient.SpecsV1alpha4().HTTPRouteGroups(destName).Delete(context.TODO(), httpRG.Name, metav1.DeleteOptions{})).To(Succeed())

		// Expect client not to reach server
		cond = Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)

			// Curl exit code 7 == Conn refused
			if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
				Td.T.Logf("> (%s) HTTP Req failed, incorrect expected result: %d, %v", srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req failed correctly: %v", srcToDestStr, result.Err)
			return true
		}, 5, 150*time.Second)
		Expect(cond).To(BeTrue())
	})
}

func setupSource(sourceName string, withKubernetesService bool) *v1.Pod {
	var svcAccDef v1.ServiceAccount
	var podDef v1.Pod
	var svcDef v1.Service
	var err error
	if Td.ClusterOS == constants.OSWindows {
		svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
			PodName:   sourceName,
			Namespace: sourceName,
			Command:   []string{"cmd", "/c"},
			Args:      []string{"FOR /L %N IN () DO ping -n 30 127.0.0.1> nul"},
			Image:     WindowsNanoserverDockerImage,
			Ports:     []int{80},
			OS:        Td.ClusterOS,
		})
	} else {
		// Get simple Pod definitions for the client
		svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
			PodName:   sourceName,
			Namespace: sourceName,
			Command:   []string{"sleep", "365d"},
			Image:     "curlimages/curl",
			Ports:     []int{80},
			OS:        Td.ClusterOS,
		})
	}

	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(sourceName, &svcAccDef)
	Expect(err).NotTo(HaveOccurred())

	srcPod, err := Td.CreatePod(sourceName, podDef)
	Expect(err).NotTo(HaveOccurred())

	// In some cases we may want to skip the creation of a Kubernetes service for the source.
	if withKubernetesService {
		_, err = Td.CreateService(sourceName, svcDef)
		Expect(err).NotTo(HaveOccurred())
	}

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(sourceName, Td.PodDeploymentTimeout, 1, nil)).To(Succeed())

	return srcPod
}

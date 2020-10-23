package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = DescribeTier1("Test HTTP traffic from 1 pod client -> 1 pod server", func() {
	Context("SimpleClientServer", func() {
		sourceName := "client"
		destName := "server"
		var ns []string = []string{sourceName, destName}

		It("Tests HTTP traffic for client pod -> server pod", func() {
			// Install OSM
			Expect(td.InstallOSM(td.GetOSMInstallOpts())).To(Succeed())

			// Create Test NS
			for _, n := range ns {
				Expect(td.CreateNs(n, nil)).To(Succeed())
				Expect(td.AddNsToMesh(true, n)).To(Succeed())
			}

			// Get simple pod definitions for the HTTP server
			svcAccDef, podDef, svcDef := td.SimplePodApp(
				SimplePodAppDef{
					name:      destName,
					namespace: destName,
					image:     "kennethreitz/httpbin",
					ports:     []int{80},
				})

			_, err := td.CreateServiceAccount(destName, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			dstPod, err := td.CreatePod(destName, podDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = td.CreateService(destName, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(td.WaitForPodsRunningReady(destName, 90*time.Second, 1)).To(Succeed())

			// Get simple Pod definitions for the client
			svcAccDef, podDef, svcDef = td.SimplePodApp(SimplePodAppDef{
				name:      sourceName,
				namespace: sourceName,
				command:   []string{"/bin/bash", "-c", "--"},
				args:      []string{"while true; do sleep 30; done;"},
				image:     "songrgg/alpine-debug",
				ports:     []int{80},
			})

			_, err = td.CreateServiceAccount(sourceName, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			srcPod, err := td.CreatePod(sourceName, podDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = td.CreateService(sourceName, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(td.WaitForPodsRunningReady(sourceName, 90*time.Second, 1)).To(Succeed())

			By("Creating SMI policies")
			// Deploy allow rule client->server
			httpRG, trafficTarget := td.CreateSimpleAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    "routes",
					TrafficTargetName: "test-target",

					SourceNamespace:      sourceName,
					SourceSVCAccountName: sourceName,

					DestinationNamespace:      destName,
					DestinationSvcAccountName: destName,
				})

			// Configs have to be put into a monitored NS
			_, err = td.CreateHTTPRouteGroup(sourceName, httpRG)
			Expect(err).NotTo(HaveOccurred())
			_, err = td.CreateTrafficTarget(sourceName, trafficTarget)
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
			clientToServer := HTTPRequestDef{
				SourceNs:        sourceName,
				SourcePod:       srcPod.Name,
				SourceContainer: sourceName,

				Destination: fmt.Sprintf("%s.%s", dstPod.Name, dstPod.Namespace),
			}

			srcToDestStr := fmt.Sprintf("%s -> %s",
				fmt.Sprintf("%s/%s", sourceName, srcPod.Name),
				clientToServer.Destination)

			cond := td.WaitForRepeatedSuccess(func() bool {
				result := td.HTTPRequest(clientToServer)

				if result.Err != nil || result.StatusCode != 200 {
					td.T.Logf("> (%s) HTTP Req failed %d %v",
						srcToDestStr, result.StatusCode, result.Err)
					return false
				}
				td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
				return true
			}, 5, 90*time.Second)
			Expect(cond).To(BeTrue())

			By("Deleting SMI policies")
			Expect(td.smiClients.AccessClient.AccessV1alpha2().TrafficTargets(sourceName).Delete(context.TODO(), trafficTarget.Name, metav1.DeleteOptions{})).To(Succeed())
			Expect(td.smiClients.SpecClient.SpecsV1alpha3().HTTPRouteGroups(sourceName).Delete(context.TODO(), httpRG.Name, metav1.DeleteOptions{})).To(Succeed())

			// Expect client not to reach server
			cond = td.WaitForRepeatedSuccess(func() bool {
				result := td.HTTPRequest(clientToServer)

				// Curl exit code 7 == Conn refused
				if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
					td.T.Logf("> (%s) HTTP Req failed, incorrect expected result: %d, %v", srcToDestStr, result.StatusCode, result.Err)
					return false
				}
				td.T.Logf("> (%s) HTTP Req failed correctly: %v", srcToDestStr, result.Err)
				return true
			}, 5, 150*time.Second)
			Expect(cond).To(BeTrue())
		})
	})
})

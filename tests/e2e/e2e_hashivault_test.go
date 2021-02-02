package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("1 Client pod -> 1 Server pod test using Vault",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("HashivaultSimpleClientServer", func() {
			const sourceNs = "client"
			const destNs = "server"
			var ns []string = []string{sourceNs, destNs}

			It("Tests HTTP traffic for client pod -> server pod", func() {
				// Install OSM
				installOpts := Td.GetOSMInstallOpts()
				installOpts.CertManager = "vault"
				Expect(Td.InstallOSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 2)).To(Succeed())

				// Create Test NS
				for _, n := range ns {
					Expect(Td.CreateNs(n, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, n)).To(Succeed())
				}

				// Get simple pod definitions for the HTTP server
				svcAccDef, podDef, svcDef := Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "server",
						Namespace: destNs,
						Image:     "kennethreitz/httpbin",
						Ports:     []int{80},
					})

				_, err := Td.CreateServiceAccount(destNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				dstPod, err := Td.CreatePod(destNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(destNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(destNs, 60*time.Second, 1)).To(Succeed())

				// Get simple Pod definitions for the client
				svcAccDef, podDef, svcDef = Td.SimplePodApp(SimplePodAppDef{
					Name:      "client",
					Namespace: sourceNs,
					Command:   []string{"/bin/bash", "-c", "--"},
					Args:      []string{"while true; do sleep 30; done;"},
					Image:     "songrgg/alpine-debug",
					Ports:     []int{80},
				})

				_, err = Td.CreateServiceAccount(sourceNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := Td.CreatePod(sourceNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(sourceNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(sourceNs, 60*time.Second, 1)).To(Succeed())

				// Deploy allow rule client->server
				httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
					SimpleAllowPolicy{
						RouteGroupName:    "routes",
						TrafficTargetName: "test-target",

						SourceNamespace:      sourceNs,
						SourceSVCAccountName: "client",

						DestinationNamespace:      destNs,
						DestinationSvcAccountName: "server",
					})

				// Configs have to be put into a monitored NS, and osm-system can't be by cli
				_, err = Td.CreateHTTPRouteGroup(sourceNs, httpRG)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(sourceNs, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				// All ready. Expect client to reach server
				// Need to get the pod though.
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result :=
						Td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: "client", // We can do better

							Destination: fmt.Sprintf("%s.%s", dstPod.Name, dstPod.Namespace),
						})

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			})
		})
	})

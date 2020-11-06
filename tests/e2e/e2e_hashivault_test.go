package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = OSMDescribe("1 Client pod -> 1 Server pod test using Vault",
	OSMDescribeInfo{
		tier:   2,
		bucket: 1,
	},
	func() {
		Context("HashivaultSimpleClientServer", func() {
			const sourceNs = "client"
			const destNs = "server"
			var ns []string = []string{sourceNs, destNs}

			It("Tests HTTP traffic for client pod -> server pod", func() {
				// Install OSM
				installOpts := td.GetOSMInstallOpts()
				installOpts.certManager = "vault"
				Expect(td.InstallOSM(installOpts)).To(Succeed())
				Expect(td.WaitForPodsRunningReady(td.osmNamespace, 60*time.Second, 2)).To(Succeed())

				// Create Test NS
				for _, n := range ns {
					Expect(td.CreateNs(n, nil)).To(Succeed())
					Expect(td.AddNsToMesh(true, n)).To(Succeed())
				}

				// Get simple pod definitions for the HTTP server
				svcAccDef, podDef, svcDef := td.SimplePodApp(
					SimplePodAppDef{
						name:      "server",
						namespace: destNs,
						image:     "kennethreitz/httpbin",
						ports:     []int{80},
					})

				_, err := td.CreateServiceAccount(destNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				dstPod, err := td.CreatePod(destNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = td.CreateService(destNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(td.WaitForPodsRunningReady(destNs, 60*time.Second, 1)).To(Succeed())

				// Get simple Pod definitions for the client
				svcAccDef, podDef, svcDef = td.SimplePodApp(SimplePodAppDef{
					name:      "client",
					namespace: sourceNs,
					command:   []string{"/bin/bash", "-c", "--"},
					args:      []string{"while true; do sleep 30; done;"},
					image:     "songrgg/alpine-debug",
					ports:     []int{80},
				})

				_, err = td.CreateServiceAccount(sourceNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := td.CreatePod(sourceNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = td.CreateService(sourceNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(td.WaitForPodsRunningReady(sourceNs, 60*time.Second, 1)).To(Succeed())

				// Deploy allow rule client->server
				httpRG, trafficTarget := td.CreateSimpleAllowPolicy(
					SimpleAllowPolicy{
						RouteGroupName:    "routes",
						TrafficTargetName: "test-target",

						SourceNamespace:      sourceNs,
						SourceSVCAccountName: "client",

						DestinationNamespace:      destNs,
						DestinationSvcAccountName: "server",
					})

				// Configs have to be put into a monitored NS, and osm-system can't be by cli
				_, err = td.CreateHTTPRouteGroup(sourceNs, httpRG)
				Expect(err).NotTo(HaveOccurred())
				_, err = td.CreateTrafficTarget(sourceNs, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				// All ready. Expect client to reach server
				// Need to get the pod though.
				cond := td.WaitForRepeatedSuccess(func() bool {
					result :=
						td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: "client", // We can do better

							Destination: fmt.Sprintf("%s.%s", dstPod.Name, dstPod.Namespace),
						})

					if result.Err != nil || result.StatusCode != 200 {
						td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			})
		})
	})

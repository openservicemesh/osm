package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = OSMDescribe("Permissive Traffic Policy Mode",
	OSMDescribeInfo{
		tier:   1,
		bucket: 2,
	},
	func() {
		Context("PermissiveMode", func() {
			const sourceNs = "client"
			const destNs = "server"
			var ns []string = []string{sourceNs, destNs}

			It("Tests HTTP traffic for client pod -> server pod with permissive mode", func() {
				// Install OSM
				installOpts := td.GetOSMInstallOpts()
				installOpts.enablePermissiveMode = true
				Expect(td.InstallOSM(installOpts)).To(Succeed())

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

				Expect(td.WaitForPodsRunningReady(destNs, 90*time.Second, 1)).To(Succeed())

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

				Expect(td.WaitForPodsRunningReady(sourceNs, 90*time.Second, 1)).To(Succeed())

				req := HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: "client",

					Destination: fmt.Sprintf("%s.%s", dstPod.Name, dstPod.Namespace),
				}

				By("Ensuring traffic is allowed when permissive mode is enabled")

				cond := td.WaitForRepeatedSuccess(func() bool {
					result := td.HTTPRequest(req)

					if result.Err != nil || result.StatusCode != 200 {
						td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())

				By("Ensuring traffic is not allowed when permissive mode is disabled")

				Expect(td.UpdateOSMConfig("permissive_traffic_policy_mode", "false"))

				cond = td.WaitForRepeatedSuccess(func() bool {
					result := td.HTTPRequest(req)

					if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
						td.T.Logf("> REST req received unexpected response (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					td.T.Logf("> REST req succeeded, got expected error: %v", result.Err)
					return true
				}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			})
		})
	})

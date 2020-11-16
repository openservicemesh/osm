package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = OSMDescribe("Test Debug Server by toggling enableDebugServer",
	OSMDescribeInfo{
		tier:   2,
		bucket: 2,
	},
	func() {
		Context("DebugServer", func() {
			const sourceNs = "client"

			It("Starts debug server only when enableDebugServer flag is enabled", func() {
				// Install OSM
				installOpts := td.GetOSMInstallOpts()
				installOpts.enableDebugServer = false
				Expect(td.InstallOSM(installOpts)).To(Succeed())

				// Create Test NS
				Expect(td.CreateNs(sourceNs, nil)).To(Succeed())

				// Get simple Pod definitions for the client
				svcAccDef, podDef, svcDef := td.SimplePodApp(SimplePodAppDef{
					name:      "client",
					namespace: sourceNs,
					command:   []string{"/bin/bash", "-c", "--"},
					args:      []string{"while true; do sleep 30; done;"},
					image:     "songrgg/alpine-debug",
					ports:     []int{80},
				})

				_, err := td.CreateServiceAccount(sourceNs, &svcAccDef)
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

					Destination: "osm-controller.osm-system:9092/debug",
				}

				By("Ensuring debug server is available when enableDebugServer is enabled")

				Expect(td.UpdateOSMConfig("enable_debug_server", "true"))

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

				By("Ensuring debug server is unavailable when enableDebugServer is disabled")

				Expect(td.UpdateOSMConfig("enable_debug_server", "false"))
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

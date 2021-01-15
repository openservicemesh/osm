package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test Debug Server by toggling enableDebugServer",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 2,
	},
	func() {
		Context("DebugServer", func() {
			const sourceNs = "client"

			It("Starts debug server only when enableDebugServer flag is enabled", func() {
				// Install OSM
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnableDebugServer = false
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				// Create Test NS
				Expect(Td.CreateNs(sourceNs, nil)).To(Succeed())

				// Get simple Pod definitions for the client
				svcAccDef, podDef, svcDef := Td.SimplePodApp(SimplePodAppDef{
					Name:      "client",
					Namespace: sourceNs,
					Command:   []string{"/bin/bash", "-c", "--"},
					Args:      []string{"while true; do sleep 30; done;"},
					Image:     "songrgg/alpine-debug",
					Ports:     []int{80},
				})

				_, err := Td.CreateServiceAccount(sourceNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := Td.CreatePod(sourceNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(sourceNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(sourceNs, 90*time.Second, 1)).To(Succeed())

				controllerDest := "osm-controller." + Td.OsmNamespace + ":9092/debug"

				req := HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: "client",

					Destination: controllerDest,
				}

				iterations := 2
				for i := 1; i <= iterations; i++ {
					By(fmt.Sprintf("(%d/%d) Ensuring debug server is available when enableDebugServer is enabled", i, iterations))

					Expect(Td.UpdateOSMConfig("enable_debug_server", "true"))

					cond := Td.WaitForRepeatedSuccess(func() bool {
						result := Td.HTTPRequest(req)

						if result.Err != nil || result.StatusCode != 200 {
							Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
							return false
						}
						Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
						return true
					}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())

					By(fmt.Sprintf("(%d/%d) Ensuring debug server is unavailable when enableDebugServer is disabled", i, iterations))

					Expect(Td.UpdateOSMConfig("enable_debug_server", "false"))

					cond = Td.WaitForRepeatedSuccess(func() bool {
						result := Td.HTTPRequest(req)

						if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
							Td.T.Logf("> REST req received unexpected response (status: %d) %v", result.StatusCode, result.Err)
							return false
						}
						Td.T.Logf("> REST req succeeded, got expected error: %v", result.Err)
						return true
					}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())
				}
			})
		})
	})

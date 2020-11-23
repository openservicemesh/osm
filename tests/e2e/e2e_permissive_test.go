package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Permissive Traffic Policy Mode",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("PermissiveMode", func() {
			const sourceNs = "client"
			const destNs = "server"
			var ns []string = []string{sourceNs, destNs}

			It("Tests HTTP traffic for client pod -> server pod with permissive mode", func() {
				// Install OSM
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnablePermissiveMode = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

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

				Expect(Td.WaitForPodsRunningReady(destNs, 90*time.Second, 1)).To(Succeed())

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

				Expect(Td.WaitForPodsRunningReady(sourceNs, 90*time.Second, 1)).To(Succeed())

				req := HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: "client",

					Destination: fmt.Sprintf("%s.%s", dstPod.Name, dstPod.Namespace),
				}

				By("Ensuring traffic is allowed when permissive mode is enabled")

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

				By("Ensuring traffic is not allowed when permissive mode is disabled")

				Expect(Td.UpdateOSMConfig("permissive_traffic_policy_mode", "false"))

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
			})
		})
	})

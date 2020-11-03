package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = OSMDescribe("HTTP and HTTPS Egress",
	OSMDescribeInfo{
		tier:   1,
		bucket: 1,
	},
	func() {
		Context("Egress", func() {
			const sourceNs = "client"

			It("Allows egress traffic when enabled", func() {
				// Install OSM
				installOpts := td.GetOSMInstallOpts()
				installOpts.egressEnabled = true
				Expect(td.InstallOSM(installOpts)).To(Succeed())

				// Create Test NS
				Expect(td.CreateNs(sourceNs, nil)).To(Succeed())
				Expect(td.AddNsToMesh(true, sourceNs)).To(Succeed())

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

				// Expect it to be up and running in it's receiver namespace
				Expect(td.WaitForPodsRunningReady(sourceNs, 60*time.Second, 1)).To(Succeed())

				protocols := []string{
					"http://",
					"https://",
				}
				egressURLs := []string{
					"edition.cnn.com",
					"github.com",
				}
				var urls []string
				for _, protocol := range protocols {
					for _, test := range egressURLs {
						urls = append(urls, protocol+test)
					}
				}

				for _, url := range urls {
					cond := td.WaitForRepeatedSuccess(func() bool {
						result := td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: "client",

							Destination: url,
						})

						if result.Err != nil || result.StatusCode != 200 {
							td.T.Logf("%s > REST req failed (status: %d) %v", url, result.StatusCode, result.Err)
							return false
						}
						td.T.Logf("%s > REST req succeeded: %d", url, result.StatusCode)
						return true
					}, 5, 60*time.Second)
					Expect(cond).To(BeTrue())
				}

				By("Disabling Egress")

				err = td.UpdateOSMConfig("egress", "false")
				Expect(err).NotTo(HaveOccurred())

				for _, url := range urls {
					cond := td.WaitForRepeatedSuccess(func() bool {
						result := td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: "client",

							Destination: url,
						})

						if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
							td.T.Logf("%s > REST req failed incorrectly (status: %d) %v", url, result.StatusCode, result.Err)
							return false
						}
						td.T.Logf("%s > REST req failed correctly: %v", url, result.Err)
						return true
					}, 5 /*success count threshold*/, 60*time.Second /*timeout*/)
					Expect(cond).To(BeTrue())
				}
			})
		})
	})

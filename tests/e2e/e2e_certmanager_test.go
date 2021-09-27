package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("1 Client pod -> 1 Server pod test using cert-manager",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 1,
	},
	func() {
		Context("CertManagerSimpleClientServer", func() {
			var (
				clientNamespace     = framework.RandomNameWithPrefix("client")
				serverNamespace     = framework.RandomNameWithPrefix("server")
				clientContainerName = framework.RandomNameWithPrefix("container")
				ns                  = []string{clientNamespace, serverNamespace}
			)

			It("Tests HTTP traffic for client pod -> server pod", func() {
				// Install OSM
				installOpts := Td.GetOSMInstallOpts()
				installOpts.CertManager = "cert-manager"
				installOpts.SetOverrides = []string{
					// increase timeout when using an external certificate provider due to
					// potential slowness issuing certs
					"OpenServiceMesh.injector.webhookTimeoutSeconds=30",
				}
				Expect(Td.InstallOSM(installOpts)).To(Succeed())
				Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 5 /* 3 cert-manager pods, 1 controller, 1 injector */, nil)).To(Succeed())

				// Create namespaces
				for _, n := range ns {
					Expect(Td.CreateNs(n, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, n)).To(Succeed())
				}

				// Get simple pod definitions for the HTTP server
				serverSvcAccDef, serverPodDef, serverSvcDef, err := Td.SimplePodApp(
					SimplePodAppDef{
						PodName:   framework.RandomNameWithPrefix("pod"),
						Namespace: serverNamespace,
						Image:     "kennethreitz/httpbin",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(serverNamespace, &serverSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				dstPod, err := Td.CreatePod(serverNamespace, serverPodDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(serverNamespace, serverSvcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(serverNamespace, 60*time.Second, 1, nil)).To(Succeed())

				// Get simple Pod definitions for the client
				clientSvcAccDef, clientPodDef, clientSvcDef, err := Td.SimplePodApp(SimplePodAppDef{
					PodName:       framework.RandomNameWithPrefix("pod"),
					Namespace:     clientNamespace,
					ContainerName: clientContainerName,
					Command:       []string{"/bin/bash", "-c", "--"},
					Args:          []string{"while true; do sleep 30; done;"},
					Image:         "songrgg/alpine-debug",
					Ports:         []int{80},
					OS:            Td.ClusterOS,
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(clientNamespace, &clientSvcAccDef)
				Expect(err).NotTo(HaveOccurred())
				srcPod, err := Td.CreatePod(clientNamespace, clientPodDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(clientNamespace, clientSvcDef)
				Expect(err).NotTo(HaveOccurred())

				// Expect it to be up and running in it's receiver namespace
				Expect(Td.WaitForPodsRunningReady(clientNamespace, 60*time.Second, 1, nil)).To(Succeed())

				// Deploy allow rule client->server
				httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
					SimpleAllowPolicy{
						RouteGroupName:    "routes",
						TrafficTargetName: "test-target",

						SourceNamespace:      clientNamespace,
						SourceSVCAccountName: clientSvcAccDef.Name,

						DestinationNamespace:      serverNamespace,
						DestinationSvcAccountName: serverSvcAccDef.Name,
					})

				// Configs have to be put into a monitored NS, and osm-system can't be by cli
				_, err = Td.CreateHTTPRouteGroup(serverNamespace, httpRG)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateTrafficTarget(serverNamespace, trafficTarget)
				Expect(err).NotTo(HaveOccurred())

				// All ready. Expect client to reach server
				// Need to get the pod though.
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result :=
						Td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: clientContainerName,

							Destination: fmt.Sprintf("%s.%s", serverSvcDef.Name, dstPod.Namespace),
						})

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 90*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			})
		})
	})

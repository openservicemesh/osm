package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = OSMDescribe("Test HTTP traffic from N deployment client -> 1 deployment server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		const (
			// to name the header we will use to identify the server that replies
			HTTPHeaderName = "podname"
		)

		var maxTestDuration = 150 * time.Second

		Context("DeploymentsClientServer", func() {
			var (
				destApp           = "server"
				sourceAppBaseName = "client"
				sourceNamespaces  = []string{}

				// Total (numberOfClientApps x replicaSetPerApp) pods
				numberOfClientServices = 5
				replicaSetPerService   = 5
			)

			// Used across the test to wait for concurrent steps to finish
			var wg sync.WaitGroup

			for i := 0; i < numberOfClientServices; i++ {
				// base names for each client service
				sourceNamespaces = append(sourceNamespaces, fmt.Sprintf("%s%d", sourceAppBaseName, i))
			}

			It("Tests HTTP traffic from multiple client deployments to a server deployment", func() {
				// Install OSM
				Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

				// Server NS
				Expect(Td.CreateNs(destApp, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, destApp)).To(Succeed())

				// Client Applications
				for _, srcClient := range sourceNamespaces {
					Expect(Td.CreateNs(srcClient, nil)).To(Succeed())
					Expect(Td.AddNsToMesh(true, srcClient)).To(Succeed())
				}

				// Use a deployment with multiple replicaset at serverside
				svcAccDef, deploymentDef, svcDef, err := Td.SimpleDeploymentApp(
					SimpleDeploymentAppDef{
						DeploymentName:     destApp,
						Namespace:          destApp,
						ServiceName:        destApp,
						ServiceAccountName: destApp,
						ReplicaCount:       int32(replicaSetPerService),
						Image:              "simonkowallik/httpbin",
						Ports:              []int{DefaultUpstreamServicePort},
						Command:            HttpbinCmd,
						OS:                 Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				// Expose an env variable such as XHTTPBIN_X_POD_NAME:
				// This httpbin fork will pick certain env variable formats and reply the values as headers.
				// We will expose pod name as one of these env variables, and will use it
				// to identify the pod that replies to the request, and validate the test
				deploymentDef.Spec.Template.Spec.Containers[0].Env = []v1.EnvVar{
					{
						Name: fmt.Sprintf("XHTTPBIN_%s", HTTPHeaderName),
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								FieldPath: "metadata.name",
							},
						},
					},
				}

				_, err = Td.CreateServiceAccount(destApp, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(destApp, deploymentDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(destApp, svcDef)
				Expect(err).NotTo(HaveOccurred())

				wg.Add(1)
				go func(wg *sync.WaitGroup, srcClient string) {
					defer GinkgoRecover()
					defer wg.Done()
					Expect(Td.WaitForPodsRunningReady(destApp, 200*time.Second, replicaSetPerService, nil)).To(Succeed())
				}(&wg, destApp)

				// Create all client deployments, also with replicaset
				for _, srcClient := range sourceNamespaces {
					svcAccDef, deploymentDef, svcDef, err = Td.SimpleDeploymentApp(
						SimpleDeploymentAppDef{
							DeploymentName:     srcClient,
							Namespace:          srcClient,
							ServiceAccountName: srcClient,
							ServiceName:        srcClient,
							ContainerName:      srcClient,
							ReplicaCount:       int32(replicaSetPerService),
							Command:            []string{"/bin/bash", "-c", "--"},
							Args:               []string{"while true; do sleep 30; done;"},
							Image:              "songrgg/alpine-debug",
							Ports:              []int{DefaultUpstreamServicePort}, // Can't deploy services with empty/no ports
							OS:                 Td.ClusterOS,
						})
					Expect(err).NotTo(HaveOccurred())

					_, err = Td.CreateServiceAccount(srcClient, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateDeployment(srcClient, deploymentDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(srcClient, svcDef)
					Expect(err).NotTo(HaveOccurred())

					wg.Add(1)
					go func(wg *sync.WaitGroup, srcClient string) {
						defer GinkgoRecover()
						defer wg.Done()
						Expect(Td.WaitForPodsRunningReady(srcClient, 200*time.Second, replicaSetPerService, nil)).To(Succeed())
					}(&wg, srcClient)
				}
				wg.Wait()

				// Create traffic rules
				// Deploy allow rules (10x)clients -> server
				for _, srcClient := range sourceNamespaces {
					httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
						SimpleAllowPolicy{
							RouteGroupName:    srcClient,
							TrafficTargetName: srcClient,

							SourceNamespace:      srcClient,
							SourceSVCAccountName: srcClient,

							DestinationNamespace:      destApp,
							DestinationSvcAccountName: destApp,
						})

					// Configs have to be put into a monitored NS, and osm-system can't be by cli
					_, err = Td.CreateHTTPRouteGroup(destApp, httpRG)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(destApp, trafficTarget)
					Expect(err).NotTo(HaveOccurred())
				}

				// Create Multiple HTTP request structure and fill it with appropriate details
				requests := HTTPMultipleRequest{
					Sources: []HTTPRequestDef{},
				}
				for _, ns := range sourceNamespaces {
					pods, err := Td.Client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
					Expect(err).To(BeNil())

					for _, pod := range pods.Items {
						requests.Sources = append(requests.Sources, HTTPRequestDef{
							SourceNs:        ns,
							SourcePod:       pod.Name,
							SourceContainer: ns, // container_name == NS for this test

							Destination: fmt.Sprintf("%s.%s:%d", destApp, destApp, DefaultUpstreamServicePort),
						})
					}
				}

				var results HTTPMultipleResults
				var serversSeen map[string]bool = map[string]bool{} // Just counts unique servers seen
				success := Td.WaitForRepeatedSuccess(func() bool {
					// Issue all calls, get results
					results = Td.MultipleHTTPRequest(&requests)

					// Care, depending on variables there could be a lot of results
					Td.PrettyPrintHTTPResults(&results)

					// Verify success conditions
					for _, ns := range results {
						for _, podResult := range ns {
							if podResult.Err != nil || podResult.StatusCode != 200 {
								return false
							}
							// We should see pod header populated
							dstPod, ok := podResult.Headers[HTTPHeaderName]
							if ok {
								// Store and mark that we have seen a response for this server pod
								serversSeen[dstPod] = true
							}
						}
					}
					Td.T.Logf("Unique servers replied %d/%d", len(serversSeen), replicaSetPerService)
					Expect(len(serversSeen)).To(Equal(replicaSetPerService))

					return true
				}, 5, maxTestDuration)

				Expect(success).To(BeTrue())
			})
		})
	})

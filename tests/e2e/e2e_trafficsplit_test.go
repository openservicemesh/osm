package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test HTTP from N Clients deployments to 1 Server deployment backed with Traffic split test",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("ClientServerTrafficSplit", func() {
			const (
				// to name the header we will use to identify the server that replies
				HTTPHeaderName = "podname"
			)
			clientAppBaseName := "client"
			serverNamespace := "server"
			trafficSplitName := "traffic-split"

			// Scale number of client services/pods here
			numberOfClientServices := 2
			clientReplicaSet := 5

			// Scale number of server services/pods here
			numberOfServerServices := 5
			serverReplicaSet := 2

			clientServices := []string{}
			serverServices := []string{}
			allNamespaces := []string{}

			for i := 0; i < numberOfClientServices; i++ {
				clientServices = append(clientServices, fmt.Sprintf("%s%d", clientAppBaseName, i))
			}

			for i := 0; i < numberOfServerServices; i++ {
				serverServices = append(serverServices, fmt.Sprintf("%s%d", serverNamespace, i))
			}

			allNamespaces = append(allNamespaces, clientServices...)
			allNamespaces = append(allNamespaces, serverNamespace) // 1 namespace for all server services (for the trafficsplit)

			// Used across the test to wait for concurrent steps to finish
			var wg sync.WaitGroup

			It("Tests HTTP traffic from Clients to the traffic split Cluster IP", func() {
				// Install OSM
				Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())

				// Create NSs
				Expect(Td.CreateMultipleNs(allNamespaces...)).To(Succeed())
				Expect(Td.AddNsToMesh(true, allNamespaces...)).To(Succeed())

				// Create server apps
				for _, serverApp := range serverServices {
					svcAccDef, deploymentDef, svcDef := Td.SimpleDeploymentApp(
						SimpleDeploymentAppDef{
							Name:         serverApp,
							Namespace:    serverNamespace,
							ReplicaCount: int32(serverReplicaSet),
							Image:        "simonkowallik/httpbin",
							Ports:        []int{80},
						})

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

					_, err := Td.CreateServiceAccount(serverNamespace, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateDeployment(serverNamespace, deploymentDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(serverNamespace, svcDef)
					Expect(err).NotTo(HaveOccurred())
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					Expect(Td.WaitForPodsRunningReady(serverNamespace, 200*time.Second, numberOfServerServices*serverReplicaSet)).To(Succeed())
				}()

				// Client apps
				for _, clientApp := range clientServices {
					svcAccDef, deploymentDef, svcDef := Td.SimpleDeploymentApp(
						SimpleDeploymentAppDef{
							Name:         clientApp,
							Namespace:    clientApp,
							ReplicaCount: int32(clientReplicaSet),
							Command:      []string{"/bin/bash", "-c", "--"},
							Args:         []string{"while true; do sleep 30; done;"},
							Image:        "songrgg/alpine-debug",
							Ports:        []int{80},
						})

					_, err := Td.CreateServiceAccount(clientApp, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateDeployment(clientApp, deploymentDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(clientApp, svcDef)
					Expect(err).NotTo(HaveOccurred())

					wg.Add(1)
					go func(app string) {
						defer wg.Done()
						Expect(Td.WaitForPodsRunningReady(app, 200*time.Second, clientReplicaSet)).To(Succeed())
					}(clientApp)
				}

				wg.Wait()

				// Put allow traffic target rules
				for _, srcClient := range clientServices {
					for _, dstServer := range serverServices {
						httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
							SimpleAllowPolicy{
								RouteGroupName:    fmt.Sprintf("%s-%s", srcClient, dstServer),
								TrafficTargetName: fmt.Sprintf("%s-%s", srcClient, dstServer),

								SourceNamespace:      srcClient,
								SourceSVCAccountName: srcClient,

								DestinationNamespace:      serverNamespace,
								DestinationSvcAccountName: dstServer,
							})

						_, err := Td.CreateHTTPRouteGroup(srcClient, httpRG)
						Expect(err).NotTo(HaveOccurred())
						_, err = Td.CreateTrafficTarget(srcClient, trafficTarget)
						Expect(err).NotTo(HaveOccurred())
					}
				}

				// Create traffic split service. Use simple Pod to create a simple service definition
				_, _, trafficSplitService := Td.SimplePodApp(SimplePodAppDef{
					Name:      trafficSplitName,
					Namespace: serverNamespace,
					Ports:     []int{80},
				})

				// Creating trafficsplit service in K8s
				_, err := Td.CreateService(serverNamespace, trafficSplitService)
				Expect(err).NotTo(HaveOccurred())

				// Create Traffic split with all server processes as backends
				trafficSplit := TrafficSplitDef{
					Name:                    trafficSplitName,
					Namespace:               serverNamespace,
					TrafficSplitServiceName: trafficSplitName,
					Backends:                []TrafficSplitBackend{},
				}
				assignation := 100 / len(serverServices) // Spreading equitatively
				for _, dstServer := range serverServices {
					trafficSplit.Backends = append(trafficSplit.Backends,
						TrafficSplitBackend{
							Name:   dstServer,
							Weight: assignation,
						},
					)
				}
				// Get the Traffic split structures
				tSplit, err := Td.CreateSimpleTrafficSplit(trafficSplit)
				Expect(err).To(BeNil())

				// Push them in K8s
				_, err = Td.CreateTrafficSplit(serverNamespace, tSplit)
				Expect(err).To(BeNil())

				By("Issuing http requests from clients to the traffic split FQDN")

				// Test traffic
				// Create Multiple HTTP request structure
				requests := HTTPMultipleRequest{
					Sources: []HTTPRequestDef{},
				}
				for _, ns := range clientServices {
					pods, err := Td.Client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
					Expect(err).To(BeNil())

					for _, pod := range pods.Items {
						requests.Sources = append(requests.Sources, HTTPRequestDef{
							SourceNs:        ns,
							SourcePod:       pod.Name,
							SourceContainer: ns, // container_name == NS for this test

							// Targeting the trafficsplit FQDN
							Destination: fmt.Sprintf("%s.%s", trafficSplitName, serverNamespace),
						})
					}
				}

				var results HTTPMultipleResults
				var serversSeen map[string]bool = map[string]bool{} // Just counts unique servers seen
				success := Td.WaitForRepeatedSuccess(func() bool {
					curlSuccess := true

					// Get results
					results = Td.MultipleHTTPRequest(&requests)

					// Print results
					Td.PrettyPrintHTTPResults(&results)

					// Verify REST status code results
					for _, ns := range results {
						for _, podResult := range ns {
							if podResult.Err != nil || podResult.StatusCode != 200 {
								curlSuccess = false
							} else {
								// We should see pod header populated
								dstPod, ok := podResult.Headers[HTTPHeaderName]
								if ok {
									// Store and mark that we have seen a response for this server pod
									serversSeen[dstPod] = true
								}
							}
						}
					}
					Td.T.Logf("Unique servers replied %d/%d",
						len(serversSeen), numberOfServerServices*serverReplicaSet)

					// Success conditions:
					// - All clients have been answered consecutively 5 successful HTTP requests
					// - We have seen all servers from the traffic split reply at least once
					return curlSuccess && (len(serversSeen) == numberOfServerServices*serverReplicaSet)
				}, 5, 150*time.Second)

				Expect(success).To(BeTrue())

				By("Issuing http requests from clients to the allowed individual service backends")

				// Test now against the individual services, observe they should still be reachable
				requests = HTTPMultipleRequest{
					Sources: []HTTPRequestDef{},
				}
				for _, clientNs := range clientServices {
					pods, err := Td.Client.CoreV1().Pods(clientNs).List(context.Background(), metav1.ListOptions{})
					Expect(err).To(BeNil())
					// For each client pod
					for _, pod := range pods.Items {
						// reach each service
						for _, svcNs := range serverServices {
							requests.Sources = append(requests.Sources, HTTPRequestDef{
								SourceNs:        pod.Namespace,
								SourcePod:       pod.Name,
								SourceContainer: pod.Namespace, // We generally code it like so for test purposes

								// direct traffic target against the specific server service in the server namespace
								Destination: fmt.Sprintf("%s.%s", svcNs, serverNamespace),
							})
						}
					}
				}

				results = HTTPMultipleResults{}
				success = Td.WaitForRepeatedSuccess(func() bool {
					// Get results
					results = Td.MultipleHTTPRequest(&requests)

					// Print results
					Td.PrettyPrintHTTPResults(&results)

					// Verify REST status code results
					for _, ns := range results {
						for _, podResult := range ns {
							if podResult.Err != nil || podResult.StatusCode != 200 {
								return false
							}
						}
					}
					return true
				}, 2, 150*time.Second)

				Expect(success).To(BeTrue())
			})
		})
	})

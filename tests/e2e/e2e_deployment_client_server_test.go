package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openservicemesh/osm/tests/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = OSMDescribe("Test HTTP traffic from N deployment client -> 1 deployment server",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 1,
	},
	func() {
		Context("DeploymentsClientServer", func() {
			const destApp = "server"
			const sourceAppBaseName = "client"
			var sourceNamespaces []string = []string{}

			// Total (numberOfClientApps x replicaSetPerApp) pods
			numberOfClientServices := 5
			replicaSetPerService := 5

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
				svcAccDef, deploymentDef, svcDef := Td.SimpleDeploymentApp(
					SimpleDeploymentAppDef{
						Name:         "server",
						Namespace:    destApp,
						ReplicaCount: int32(replicaSetPerService),
						Image:        "kennethreitz/httpbin",
						Ports:        []int{80},
					})

				_, err := Td.CreateServiceAccount(destApp, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(destApp, deploymentDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(destApp, svcDef)
				Expect(err).NotTo(HaveOccurred())

				wg.Add(1)
				go func(wg *sync.WaitGroup, srcClient string) {
					defer wg.Done()
					Expect(Td.WaitForPodsRunningReady(destApp, 200*time.Second, replicaSetPerService)).To(Succeed())
				}(&wg, destApp)

				// Create all client deployments, also with replicaset
				for _, srcClient := range sourceNamespaces {
					svcAccDef, deploymentDef, svcDef = Td.SimpleDeploymentApp(
						SimpleDeploymentAppDef{
							Name:         srcClient,
							Namespace:    srcClient,
							ReplicaCount: int32(replicaSetPerService),
							Command:      []string{"/bin/bash", "-c", "--"},
							Args:         []string{"while true; do sleep 30; done;"},
							Image:        "songrgg/alpine-debug",
							Ports:        []int{80}, // Can't deploy services with empty/no ports
						})
					_, err = Td.CreateServiceAccount(srcClient, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateDeployment(srcClient, deploymentDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(srcClient, svcDef)
					Expect(err).NotTo(HaveOccurred())

					wg.Add(1)
					go func(wg *sync.WaitGroup, srcClient string) {
						defer wg.Done()
						Expect(Td.WaitForPodsRunningReady(srcClient, 200*time.Second, replicaSetPerService)).To(Succeed())
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
					_, err = Td.CreateHTTPRouteGroup(srcClient, httpRG)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(srcClient, trafficTarget)
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

							Destination: fmt.Sprintf("%s.%s", destApp, destApp),
						})
					}
				}

				var results HTTPMultipleResults
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
						}
					}
					return true
				}, 5, 150*time.Second)

				Expect(success).To(BeTrue())
			})
		})
	})

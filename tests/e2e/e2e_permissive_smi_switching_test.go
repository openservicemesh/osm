package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = OSMDescribe("Test HTTP traffic from N deployment client -> 1 deployment server, permissive to SMI switching",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("PermissiveToSmiSwitching", func() {
			var (
				destApp           = framework.RandomNameWithPrefix("server")
				sourceAppBaseName = framework.RandomNameWithPrefix("client")
				sourceNamespaces  = []string{}

				// Total (numberOfClientApps x replicaSetPerApp) pods
				numberOfClientServices = 2
				replicaSetPerService   = 2
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
						ServiceAccountName: destApp,
						ServiceName:        destApp,
						ReplicaCount:       int32(replicaSetPerService),
						Image:              "kennethreitz/httpbin",
						Ports:              []int{DefaultUpstreamServicePort},
						Command:            HttpbinCmd,
						OS:                 Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(destApp, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateDeployment(destApp, deploymentDef)
				Expect(err).NotTo(HaveOccurred())
				serverService, err := Td.CreateService(destApp, svcDef)
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
							SourceContainer: ns,

							Destination: fmt.Sprintf("%s.%s:%d", serverService.Name, serverService.Namespace, DefaultUpstreamServicePort),
						})
					}
				}

				By("Fails traffic test with no permissive mode and no SMI rules")
				Expect(trafficTest(false, requests)).To(BeTrue())

				By("Succeeds traffic test when permissive mode is enabled between inmesh targets")
				Expect(setPermissiveMode(true)).To(BeNil())
				Expect(trafficTest(true, requests)).To(BeTrue())

				// ---
				// The following two steps are added to test, debug and make sure unsubscriptions happen at XDS level
				// They may seem redundant; they are not.
				By("Fails traffic test again again when permissive is disabled")
				Expect(setPermissiveMode(false)).To(BeNil())
				Expect(trafficTest(false, requests)).To(BeTrue())

				By("Traffic succeeds again when re-enabled")
				Expect(setPermissiveMode(true)).To(BeNil())
				Expect(trafficTest(true, requests)).To(BeTrue())
				// ---

				By("Traffic succeeds again when re-enabled")
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

					_, err = Td.CreateHTTPRouteGroup(destApp, httpRG)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateTrafficTarget(destApp, trafficTarget)
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(trafficTest(true, requests)).To(BeTrue())

				By("Succeeds when disabling Permissive now that SMI rules are in place")
				Expect(setPermissiveMode(false)).To(BeNil())
				Expect(trafficTest(true, requests)).To(BeTrue())
			})
		})
	})

func setPermissiveMode(b bool) error {
	meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
	if err != nil {
		return err
	}
	meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode = b
	_, err = Td.UpdateOSMConfig(meshConfig)
	return err
}

func trafficTest(expectPass bool, requests HTTPMultipleRequest) bool {
	return Td.WaitForRepeatedSuccess(func() bool {
		// Issue all calls, get results
		results := Td.MultipleHTTPRequest(&requests)

		// Care, depending on variables there could be a lot of results
		Td.PrettyPrintHTTPResults(&results)

		// Verify success conditions
		for _, ns := range results {
			for _, podResult := range ns {
				if expectPass {
					if podResult.Err != nil || podResult.StatusCode != 200 {
						return false
					}
				} else {
					if podResult.StatusCode == 200 {
						return false
					}
				}
			}
		}
		return true
	}, 15, 200*time.Second)
}

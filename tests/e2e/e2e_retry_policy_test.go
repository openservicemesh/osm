package e2e

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
	. "github.com/openservicemesh/osm/tests/framework"
)

const client = "client"
const server = "server"

var meshNs = []string{client, server}

var _ = OSMDescribe("Test Retry Policy",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 8,
	},
	func() {
		Context("Retry policy enabled", func() {
			It("tests retryOn and numRetries field for retry policy",
				func() {
					// Install OSM
					installOpts := Td.GetOSMInstallOpts()
					installOpts.EnablePermissiveMode = true
					installOpts.EnableRetryPolicy = true
					Expect(Td.InstallOSM(installOpts)).To(Succeed())

					// Create test NS in mesh
					for _, n := range meshNs {
						Expect(Td.CreateNs(n, nil)).To(Succeed())
						Expect(Td.AddNsToMesh(true, n)).To(Succeed())
					}

					// Load retry image
					retryImage := fmt.Sprintf("%s/retry:%s", installOpts.ContainerRegistryLoc, installOpts.OsmImagetag)
					Expect(Td.LoadImagesToKind([]string{"retry"})).To(Succeed())

					svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
						SimplePodAppDef{
							PodName:            server,
							Namespace:          server,
							ServiceAccountName: server,
							Command:            []string{"/retry"},
							Image:              retryImage,
							Ports:              []int{9091},
							OS:                 Td.ClusterOS,
						})
					Expect(err).NotTo(HaveOccurred())

					_, err = Td.CreateServiceAccount(server, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreatePod(server, podDef)
					Expect(err).NotTo(HaveOccurred())
					serverSvc, err := Td.CreateService(server, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(server, 90*time.Second, 1, nil)).To(Succeed())

					// Get simple Pod definitions for the source/client
					svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
						PodName:   client,
						Namespace: client,
						Command:   []string{"sleep", "365d"},
						Image:     "curlimages/curl",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
					Expect(err).NotTo(HaveOccurred())

					clientSvcAcct, err := Td.CreateServiceAccount(client, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					clientPod, err := Td.CreatePod(client, podDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(client, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(client, 90*time.Second, 1, nil)).To(Succeed())

					retry := &v1alpha1.Retry{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "retrypolicy",
							Namespace: client,
						},
						Spec: v1alpha1.RetrySpec{
							Source: v1alpha1.RetrySrcDstSpec{
								Kind:      "ServiceAccount",
								Name:      clientSvcAcct.Name,
								Namespace: client,
							},
							Destinations: []v1alpha1.RetrySrcDstSpec{
								{
									Kind:      "Service",
									Name:      serverSvc.Name,
									Namespace: server,
								},
							},
							RetryPolicy: v1alpha1.RetryPolicySpec{
								RetryOn:                  "5xx",
								PerTryTimeout:            &metav1.Duration{Duration: time.Duration(1 * time.Second)},
								NumRetries:               &NumRetries,
								RetryBackoffBaseInterval: &metav1.Duration{Duration: time.Duration(5 * time.Second)},
							},
						},
					}
					_, err = Td.PolicyClient.PolicyV1alpha1().Retries(client).Create(context.TODO(), retry, metav1.CreateOptions{})
					Expect(err).ToNot((HaveOccurred()))

					req := HTTPRequestDef{
						SourceNs:        client,
						SourcePod:       clientPod.Name,
						SourceContainer: podDef.GetName(),
						Destination:     fmt.Sprintf("%s.%s.svc.cluster.local:9091", serverSvc.Name, server),
					}

					By("A request that will be retried NumRetries times then succeed")
					// wait for server
					time.Sleep(3 * time.Second)
					result := Td.RetryHTTPRequest(req)
					// One count is the initial http request that returns a retriable status code
					// followed by numRetries retries
					Expect(result.RequestCount).To(Equal(int(NumRetries) + 1))
					Expect(result.StatusCode).To(Equal(200))
					Expect(result.Err).To(BeNil())
				})
		})
		Context("Retry policy disabled", func() {
			It("tests retry does not occur",
				func() {
					// Install OSM
					installOpts := Td.GetOSMInstallOpts()
					installOpts.EnablePermissiveMode = true
					installOpts.EnableRetryPolicy = false
					Expect(Td.InstallOSM(installOpts)).To(Succeed())

					// Create test NS in mesh
					for _, n := range meshNs {
						Expect(Td.CreateNs(n, nil)).To(Succeed())
						Expect(Td.AddNsToMesh(true, n)).To(Succeed())
					}
					// Load retry image
					retryImage := fmt.Sprintf("%s/retry:%s", installOpts.ContainerRegistryLoc, installOpts.OsmImagetag)
					Expect(Td.LoadImagesToKind([]string{"retry"})).To(Succeed())

					svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
						SimplePodAppDef{
							PodName:            server,
							Namespace:          server,
							ServiceAccountName: server,
							Command:            []string{"/retry"},
							Image:              retryImage,
							Ports:              []int{9091},
							OS:                 Td.ClusterOS,
						})
					Expect(err).NotTo(HaveOccurred())

					_, err = Td.CreateServiceAccount(server, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreatePod(server, podDef)
					Expect(err).NotTo(HaveOccurred())
					serverSvc, err := Td.CreateService(server, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(server, 90*time.Second, 1, nil)).To(Succeed())

					// Get simple Pod definitions for the source/client
					svcAccDef, podDef, svcDef, err = Td.SimplePodApp(SimplePodAppDef{
						PodName:   client,
						Namespace: client,
						Command:   []string{"sleep", "365d"},
						Image:     "curlimages/curl",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
					Expect(err).NotTo(HaveOccurred())

					clientSvcAcct, err := Td.CreateServiceAccount(client, &svcAccDef)
					Expect(err).NotTo(HaveOccurred())
					clientPod, err := Td.CreatePod(client, podDef)
					Expect(err).NotTo(HaveOccurred())
					_, err = Td.CreateService(client, svcDef)
					Expect(err).NotTo(HaveOccurred())

					Expect(Td.WaitForPodsRunningReady(client, 90*time.Second, 1, nil)).To(Succeed())

					retry := &v1alpha1.Retry{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "retrypolicy",
							Namespace: client,
						},
						Spec: v1alpha1.RetrySpec{
							Source: v1alpha1.RetrySrcDstSpec{
								Kind:      "ServiceAccount",
								Name:      clientSvcAcct.Name,
								Namespace: client,
							},
							Destinations: []v1alpha1.RetrySrcDstSpec{
								{
									Kind:      "Service",
									Name:      serverSvc.Name,
									Namespace: server,
								},
							},
							RetryPolicy: v1alpha1.RetryPolicySpec{
								RetryOn:                  "5xx",
								PerTryTimeout:            &metav1.Duration{Duration: time.Duration(1 * time.Second)},
								NumRetries:               &NumRetries,
								RetryBackoffBaseInterval: &metav1.Duration{Duration: time.Duration(5 * time.Second)},
							},
						},
					}
					_, err = Td.PolicyClient.PolicyV1alpha1().Retries(client).Create(context.TODO(), retry, metav1.CreateOptions{})
					Expect(err).ToNot((HaveOccurred()))

					req := HTTPRequestDef{
						SourceNs:        client,
						SourcePod:       clientPod.Name,
						SourceContainer: podDef.GetName(),
						Destination:     fmt.Sprintf("%s.%s.svc.cluster.local:9091", serverSvc.Name, server),
					}

					By("A request that will not be retried on")
					// wait for server
					time.Sleep(3 * time.Second)
					result := Td.RetryHTTPRequest(req)
					// One count is the initial http request that is not retried on
					Expect(result.RequestCount).To(Equal(1))
					Expect(result.StatusCode).To(Equal(555))
					Expect(result.Err).To(BeNil())
				})
		})
	})

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Custom WASM metrics between one client pod and one server",
	OSMDescribeInfo{
		Tier:   2, // experimental feature
		Bucket: 4,
	},
	func() {
		const sourceNs = "clientns"
		const destNs = "serverns"
		var ns []string = []string{sourceNs, destNs}

		It("Generates metrics with the right labels and values", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			installOpts.DeployPrometheus = true
			installOpts.SetOverrides = []string{"OpenServiceMesh.enableWASMStatsExperimental=true"}
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			// Create Test NS
			for _, n := range ns {
				Expect(Td.CreateNs(n, nil)).To(Succeed())
				Expect(Td.AddNsToMesh(true, n)).To(Succeed())
			}
			stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/osm"), []string{"metrics", "enable", "--namespace", strings.Join(ns, ",")})
			Td.T.Log(stdout)
			if err != nil {
				Td.T.Logf("stderr:\n%s", stderr)
			}
			Expect(err).NotTo(HaveOccurred())

			// Get simple pod definitions for the HTTP server
			svcAccDef, depDef, svcDef := Td.SimpleDeploymentApp(
				SimpleDeploymentAppDef{
					ReplicaCount: 1,
					Name:         "server",
					Namespace:    destNs,
					Image:        "kennethreitz/httpbin",
					Ports:        []int{80},
				})

			_, err = Td.CreateServiceAccount(destNs, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			dstDep, err := Td.CreateDeployment(destNs, depDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(destNs, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(destNs, 60*time.Second, 1)).To(Succeed())

			// Get simple Pod definitions for the client
			svcAccDef, depDef, svcDef = Td.SimpleDeploymentApp(SimpleDeploymentAppDef{
				ReplicaCount: 1,
				Name:         "client",
				Namespace:    sourceNs,
				Command:      []string{"/bin/bash", "-c", "--"},
				Args:         []string{"while true; do sleep 30; done;"},
				Image:        "songrgg/alpine-debug",
				Ports:        []int{80},
			})

			_, err = Td.CreateServiceAccount(sourceNs, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			srcDep, err := Td.CreateDeployment(sourceNs, depDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(sourceNs, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(sourceNs, 60*time.Second, 1)).To(Succeed())

			// Deploy allow rule client->server
			httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    "routes",
					TrafficTargetName: "test-target",

					SourceNamespace:      sourceNs,
					SourceSVCAccountName: "client",

					DestinationNamespace:      destNs,
					DestinationSvcAccountName: "server",
				})

			// SMI is formally deployed on destination NS
			_, err = Td.CreateHTTPRouteGroup(destNs, httpRG)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(destNs, trafficTarget)
			Expect(err).NotTo(HaveOccurred())

			srcPods, err := Td.Client.CoreV1().Pods(sourceNs).List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(srcPods.Items).To(HaveLen(1))
			srcPod := srcPods.Items[0]
			successCount := 5
			cond := Td.WaitForRepeatedSuccess(func() bool {
				result :=
					Td.HTTPRequest(HTTPRequestDef{
						SourceNs:        srcPod.Namespace,
						SourcePod:       srcPod.Name,
						SourceContainer: "client",

						Destination: fmt.Sprintf("%s.%s/status/200", dstSvc.Name, dstSvc.Namespace),
					})

				if result.Err != nil || result.StatusCode != 200 {
					Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
					return false
				}
				Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
				return true
			}, successCount, 90*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())

			pods, err := Td.Client.CoreV1().Pods(Td.OsmNamespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: "app=osm-prometheus",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).To(HaveLen(1))
			var promPort int32
			for _, c := range pods.Items[0].Spec.Containers {
				if c.Name == "prometheus" {
					promPort = c.Ports[0].ContainerPort
					break
				}
			}
			Expect(promPort).NotTo(Equal(0))

			prometheus, err := Td.GetOSMPrometheusHandle()
			Expect(err).NotTo(HaveOccurred())
			defer prometheus.Stop()

			dstPods, err := Td.Client.CoreV1().Pods(destNs).List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(dstPods.Items).To(HaveLen(1))
			dstPod := dstPods.Items[0]

			queryLabels := fmt.Sprintf(
				"source_namespace=%q,"+
					"source_name=%q,"+
					"source_pod=%q,"+
					"source_kind=\"Deployment\","+
					"destination_namespace=%q,"+
					"destination_name=%q,"+
					"destination_pod=%q,"+
					"destination_kind=\"Deployment\"",
				sourceNs,
				srcDep.Name,
				strings.ReplaceAll(srcPod.Name, "-", "_"), // proxy-wasm turns '-' into '_' for metric labels
				destNs,
				dstDep.Name,
				strings.ReplaceAll(dstPod.Name, "-", "_"),
			)

			metricsOK := func(query string) func() bool {
				return func() bool {
					Td.T.Logf("querying Prometheus: %q", query)
					queryResult, err := prometheus.VectorQuery(query, time.Now())
					if err != nil {
						Td.T.Log("error querying prometheus:", err)
						return false
					}

					Td.T.Logf("verifying query result %v", queryResult)
					if queryResult != float64(successCount) {
						Td.T.Logf("Expected value to be %v, got %v", successCount, queryResult)
						return false
					}
					Td.T.Log("metrics ok")
					return true
				}
			}

			// Due to the timing of when Prometheus scrapes metrics, we wrap
			// the checks in a retry loop in case Prometheus hasn't scraped all
			// the latest metrics.
			cond = Td.WaitForRepeatedSuccess(
				metricsOK(fmt.Sprintf(`osm_request_total{response_code="200",%s}`, queryLabels)),
				1 /*success count*/, 30*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())

			cond = Td.WaitForRepeatedSuccess(
				metricsOK(fmt.Sprintf(`osm_request_duration_ms_bucket{le="+Inf",%s}`, queryLabels)),
				1 /*success count*/, 30*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())
		})
	})

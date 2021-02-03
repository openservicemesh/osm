package e2e

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	smiAccessV1alpha2 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha2"
	smiSpecsV1alpha3 "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/specs/v1alpha3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Upgrade from latest",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 2,
	},
	func() {
		const ns = "upgrade-test"

		It("Tests upgrading the control plane", func() {
			if Td.InstType == NoInstall {
				Td.T.Skip("test requires fresh OSM install")
			}

			if _, err := exec.LookPath("kubectl"); err != nil {
				Td.T.Fatal("\"kubectl\" command required and not found on PATH")
			}

			helmCfg := &action.Configuration{}
			Expect(helmCfg.Init(Td.Env.RESTClientGetter(), Td.OsmNamespace, "secret", Td.T.Logf)).To(Succeed())
			helmEnv := cli.New()

			// Install OSM with Helm vs. CLI so the test isn't dependent on
			// multiple versions of the CLI at once.
			Expect(Td.CreateNs(Td.OsmNamespace, nil)).To(Succeed())
			const releaseName = "osm"
			i := action.NewInstall(helmCfg)

			// Latest version excluding pre-releases used by default. Using the
			// latest assumes we aren't maintaining multiple release branches
			// at once. e.g. if a patch is cut for both v0.5.0 and v0.6.0, we
			// wouldn't want to test "upgrading" backwards from v0.6.0 to
			// v0.5.1.
			i.ChartPathOptions.RepoURL = "https://openservicemesh.github.io/osm"
			i.Version = ">0.0.0-0" // Include pre-releases
			i.Namespace = Td.OsmNamespace
			i.Wait = true
			i.ReleaseName = releaseName
			i.Timeout = 120 * time.Second
			vals := map[string]interface{}{
				"OpenServiceMesh": map[string]interface{}{
					"deployPrometheus": true,
					"deployJaeger":     false,
				},
			}
			chartPath, err := i.LocateChart("osm", helmEnv)
			Expect(err).NotTo(HaveOccurred())
			ch, err := loader.Load(chartPath)
			Expect(err).NotTo(HaveOccurred())
			Td.T.Log("testing upgrade from chart version", ch.Metadata.Version)

			_, err = i.Run(ch, vals)
			Expect(err).NotTo(HaveOccurred())

			// Create Test NS
			Expect(Td.CreateNs(ns, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, ns)).To(Succeed())

			// Get simple pod definitions for the HTTP server
			svcAccDef, podDef, svcDef := Td.SimplePodApp(
				SimplePodAppDef{
					Name:      "server",
					Namespace: ns,
					Image:     "kennethreitz/httpbin",
					Ports:     []int{80},
				})

			_, err = Td.CreateServiceAccount(ns, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			dstPod, err := Td.CreatePod(ns, podDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(ns, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Get simple Pod definitions for the client
			svcAccDef, podDef, svcDef = Td.SimplePodApp(SimplePodAppDef{
				Name:      "client",
				Namespace: ns,
				Command:   []string{"/bin/bash", "-c", "--"},
				Args:      []string{"while true; do sleep 30; done;"},
				Image:     "songrgg/alpine-debug",
				Ports:     []int{80},
			})

			_, err = Td.CreateServiceAccount(ns, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			srcPod, err := Td.CreatePod(ns, podDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(ns, svcDef)
			Expect(err).NotTo(HaveOccurred())

			Expect(Td.WaitForPodsRunningReady(ns, 90*time.Second, 2)).To(Succeed())

			httpRG := smiSpecsV1alpha3.HTTPRouteGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name: "routes",
				},
				Spec: smiSpecsV1alpha3.HTTPRouteGroupSpec{
					Matches: []smiSpecsV1alpha3.HTTPMatch{
						{
							Name:      "all",
							PathRegex: ".*",
							Methods:   []string{"*"},
						},
					},
				},
			}

			trafficTarget := smiAccessV1alpha2.TrafficTarget{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-target",
				},
				Spec: smiAccessV1alpha2.TrafficTargetSpec{
					Sources: []smiAccessV1alpha2.IdentityBindingSubject{
						{
							Kind:      "ServiceAccount",
							Name:      "client",
							Namespace: ns,
						},
					},
					Destination: smiAccessV1alpha2.IdentityBindingSubject{
						Kind:      "ServiceAccount",
						Name:      "server",
						Namespace: ns,
					},
					Rules: []smiAccessV1alpha2.TrafficTargetRule{
						{
							Kind: "HTTPRouteGroup",
							Name: "routes",
							Matches: []string{
								"all",
							},
						},
					},
				},
			}

			// Configs have to be put into a monitored NS, and osm-system can't be by cli
			_, err = Td.SmiClients.SpecClient.SpecsV1alpha3().HTTPRouteGroups(ns).Create(context.Background(), &httpRG, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.SmiClients.AccessClient.AccessV1alpha2().TrafficTargets(ns).Create(context.Background(), &trafficTarget, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
			checkClientToServerOK := func() {
				By("Checking client can make requests to server")
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result :=
						Td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: "client",
							Destination:     dstPod.Name + "/status/200",
						})

					if result.Err != nil || result.StatusCode != 200 {
						Td.T.Logf("> REST req failed (status: %d) %v", result.StatusCode, result.Err)
						return false
					}
					Td.T.Logf("> REST req succeeded: %d", result.StatusCode)
					return true
				}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			}

			checkProxiesConnected := func() {
				By("Checking all proxies are connected")
				prometheus, err := Td.GetOSMPrometheusHandle()
				Expect(err).NotTo(HaveOccurred())
				defer prometheus.Stop()
				cond := Td.WaitForRepeatedSuccess(func() bool {
					expectedProxyCount := float64(2)
					proxies, err := prometheus.VectorQuery("sum(envoy_control_plane_connected_state)", time.Now())
					if err != nil {
						Td.T.Log("error querying prometheus:", err)
						return false
					}

					if proxies != expectedProxyCount {
						Td.T.Logf("expected query result %v, got %v", expectedProxyCount, proxies)
						return false
					}

					Td.T.Log("All proxies connected")
					return true
				}, 5 /*success threshold*/, 30*time.Second /*timeout*/)
				Expect(cond).To(BeTrue())
			}

			checkProxiesConnected()
			checkClientToServerOK()

			// TODO: Only delete and recreate the CRDs if needed
			By("Upgrading CRDs")

			err = Td.SmiClients.SpecClient.SpecsV1alpha3().HTTPRouteGroups(ns).Delete(context.Background(), httpRG.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
			err = Td.SmiClients.AccessClient.AccessV1alpha2().TrafficTargets(ns).Delete(context.Background(), trafficTarget.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())

			helm := &action.Configuration{}
			Expect(helm.Init(Td.Env.RESTClientGetter(), Td.OsmNamespace, "secret", Td.T.Logf)).To(Succeed())
			rel, err := action.NewGet(helm).Run(releaseName)
			Expect(err).NotTo(HaveOccurred())
			for _, crd := range rel.Chart.CRDs() {
				res, err := helm.KubeClient.Build(bytes.NewReader(crd.Data), false)
				Expect(err).NotTo(HaveOccurred())
				_, errs := helm.KubeClient.Delete(res)
				Expect(errs).To(BeNil())
			}

			// TODO: Find a decent way to do this without relying on the kubectl binary
			stdout, stderr, err := Td.RunLocal("kubectl", []string{"apply", "-f", filepath.FromSlash("../../charts/osm/crds")})
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			By("Upgrading OSM")

			if Td.InstType == KindCluster {
				Expect(Td.LoadOSMImagesIntoKind()).To(Succeed())
			}

			stdout, stderr, err = Td.RunLocal(filepath.FromSlash("../../bin/osm"), []string{"mesh", "upgrade", "--osm-namespace=" + Td.OsmNamespace, "--container-registry=" + Td.CtrRegistryServer, "--osm-image-tag=" + Td.OsmImageTag})
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			// Deploy allow rule client->server
			newHTTPRG, newTrafficTarget := Td.CreateSimpleAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    "routes",
					TrafficTargetName: "test-target",

					SourceNamespace:      ns,
					SourceSVCAccountName: "client",

					DestinationNamespace:      ns,
					DestinationSvcAccountName: "server",
				},
			)
			_, err = Td.CreateHTTPRouteGroup(ns, newHTTPRG)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(ns, newTrafficTarget)
			Expect(err).NotTo(HaveOccurred())

			checkClientToServerOK()
			checkProxiesConnected()
		})
	})

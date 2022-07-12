package e2e

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	corev1 "k8s.io/api/core/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Upgrade from latest",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 10,
	},
	func() {
		const ns = "upgrade-test"

		It("Tests upgrading the control plane", func() {
			if Td.InstType == NoInstall {
				Skip("test requires fresh OSM install")
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

			i.ChartPathOptions.RepoURL = "https://openservicemesh.github.io/osm"
			// On the main branch, this should refer to the latest release. On
			// release branches, it should specify the most recent patch of the
			// previous minor release. e.g. on the release-v1.0 branch, this
			// should be "0.11".
			i.Version = "1.1"
			i.Namespace = Td.OsmNamespace
			i.Wait = true
			i.ReleaseName = releaseName
			i.Timeout = 120 * time.Second
			vals := map[string]interface{}{
				"osm": map[string]interface{}{
					"deployPrometheus": true,
					// Init container must be privileged if an OpenShift cluster is being used
					"enablePrivilegedInitContainer": Td.DeployOnOpenShift,

					// Reduce CPU so CI (capped at 2 CPU) can handle standing
					// up the new control plane before tearing the old one
					// down.
					"osmController": map[string]interface{}{
						"resource": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu": "0.3",
							},
						},
					},
					"injector": map[string]interface{}{
						"resource": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu": "0.1",
							},
						},
					},
					"prometheus": map[string]interface{}{
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"cpu":    "0.1",
								"memory": "256M",
							},
						},
					},
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
			serverSvcAccDef, serverPodDef, serverSvcDef, err := Td.GetOSSpecificHTTPBinPod("server", ns)
			Expect(err).NotTo(HaveOccurred())

			_, err = Td.CreateServiceAccount(ns, &serverSvcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(ns, serverPodDef)
			Expect(err).NotTo(HaveOccurred())
			dstSvc, err := Td.CreateService(ns, serverSvcDef)
			Expect(err).NotTo(HaveOccurred())

			// Get simple Pod definitions for the client
			svcAccDef, srcPodDef, svcDef, err := Td.SimplePodApp(SimplePodAppDef{
				PodName:   "client",
				Namespace: ns,
				Command:   []string{"sleep", "365d"},
				Image:     "curlimages/curl",
				Ports:     []int{80},
				OS:        Td.ClusterOS,
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = Td.CreateServiceAccount(ns, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			srcPod, err := Td.CreatePod(ns, srcPodDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(ns, svcDef)
			Expect(err).NotTo(HaveOccurred())

			Expect(Td.WaitForPodsRunningReady(ns, 90*time.Second, 2, nil)).To(Succeed())

			// Deploy allow rule client->server
			httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
				SimpleAllowPolicy{
					RouteGroupName:    "routes",
					TrafficTargetName: "test-target",

					SourceNamespace:      ns,
					SourceSVCAccountName: svcAccDef.Name,

					DestinationNamespace:      ns,
					DestinationSvcAccountName: serverSvcAccDef.Name,
				},
			)
			_, err = Td.CreateHTTPRouteGroup(ns, httpRG)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateTrafficTarget(ns, trafficTarget)
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
			checkClientToServerOK := func() {
				By("Checking client can make requests to server")
				cond := Td.WaitForRepeatedSuccess(func() bool {
					result :=
						Td.HTTPRequest(HTTPRequestDef{
							SourceNs:        srcPod.Namespace,
							SourcePod:       srcPod.Name,
							SourceContainer: srcPod.Name,
							Destination:     fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace) + "/status/200",
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
					proxies, err := prometheus.VectorQuery("osm_proxy_connect_count", time.Now())
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

			By("Upgrading OSM")

			pullPolicy := corev1.PullAlways
			if Td.InstType == KindCluster {
				pullPolicy = corev1.PullIfNotPresent
				Expect(Td.LoadOSMImagesIntoKind()).To(Succeed())
			}

			setArgs := "--set=osm.image.tag=" + Td.OsmImageTag + ",osm.image.registry=" + Td.CtrRegistryServer + ",osm.image.pullPolicy=" + string(pullPolicy) + ",osm.deployPrometheus=true,osm.enablePrivilegedInitContainer=" + strconv.FormatBool(Td.DeployOnOpenShift) + ",osm.osmController.resource.requests.cpu=0.3,osm.injector.resource.requests.cpu=0.1,osm.prometheus.resources.requests.cpu=0.1,osm.prometheus.resources.requests.memory=256M"
			stdout, stderr, err := Td.RunLocal(filepath.FromSlash("../../bin/osm"), "mesh", "upgrade", "--osm-namespace="+Td.OsmNamespace, setArgs)
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			// Verify that all the CRD's required by OSM are present in the cluster post an upgrade
			// TODO: Find a decent way to do this without relying on the kubectl binary
			// TODO: In the future when we bump the version on a CRD, we need to update this check to ensure that the version is the latest required version
			stdout, stderr, err = Td.RunLocal("kubectl", "get", "-f", filepath.FromSlash("../../cmd/osm-bootstrap/crds"))
			Td.T.Log(stdout.String())
			if err != nil {
				Td.T.Log("stderr:\n" + stderr.String())
			}
			Expect(err).NotTo(HaveOccurred())

			checkClientToServerOK()
			checkProxiesConnected()
		})
	})

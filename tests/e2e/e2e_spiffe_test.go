package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("SPIFFE",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 10,
	},
	func() {
		Context("with Tressor", func() {
			testSpiffeCert()
		})
	})

func testSpiffeCert() {
	It("has a Pod Certificate with a SPIFFE ID", func() {
		By("Installing OSM with SPIFFE Enabled")
		installOpts := Td.GetOSMInstallOpts(WithSpiffeEnabled())
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		By("creating a test workload")
		clientPod, serverPod, serverSvc := deployTestWorkload()
		verifySpiffeIDInPodCert(clientPod)
		verifySpiffeIDInPodCert(serverPod)

		By("checking HTTP traffic for client -> server pod cert")
		verifySuccessfulPodConnection(clientPod, serverPod, serverSvc)
	})
}

func verifySpiffeIDInPodCert(pod *v1.Pod) {
	By("Verifying pod has SPIFFE cert in URI SAN")

	// It can take a moment for envoy to load the certs
	Eventually(func() (string, error) {
		args := []string{"proxy", "get", "certs", pod.Name, fmt.Sprintf("-n=%s", pod.Namespace)}
		stdout, _, err := Td.RunOsmCli(args...)
		Td.T.Logf("stdout:\n%s", stdout)
		return stdout.String(), err
	}, 10*time.Second).Should(ContainSubstring(fmt.Sprintf("\"uri\": \"spiffe://cluster.local/%s/%s", pod.Spec.ServiceAccountName, pod.Namespace)))
}

func verifySpiffeIDForDeployment(deployment appsv1.Deployment) {
	By("By getting pods in deployment")

	pods, err := Td.GetPodsForLabel(deployment.Namespace, *deployment.Spec.Selector)
	Expect(err).To(BeNil())

	for i := range pods {
		verifySpiffeIDInPodCert(&pods[i])
	}
}

func deployTestWorkload() (*v1.Pod, *v1.Pod, *v1.Service) {
	var (
		clientNamespace = framework.RandomNameWithPrefix("client")
		serverNamespace = framework.RandomNameWithPrefix("server")
		ns              = []string{clientNamespace, serverNamespace}
	)

	By("Deploying client -> server workload")
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
			Image:     fortioImageName,
			Ports:     []int{fortioHTTPPort},
			OS:        Td.ClusterOS,
		})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(serverNamespace, &serverSvcAccDef)
	Expect(err).NotTo(HaveOccurred())
	serverPod, err := Td.CreatePod(serverNamespace, serverPodDef)
	Expect(err).NotTo(HaveOccurred())
	serverSvc, err := Td.CreateService(serverNamespace, serverSvcDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(serverNamespace, 60*time.Second, 1, nil)).To(Succeed())

	// Get simple Pod definitions for the client
	podName := framework.RandomNameWithPrefix("pod")
	clientSvcAccDef, clientPodDef, clientSvcDef, err := Td.SimplePodApp(SimplePodAppDef{
		PodName:       podName,
		Namespace:     clientNamespace,
		ContainerName: podName,
		Image:         fortioImageName,
		Ports:         []int{fortioHTTPPort},
		OS:            Td.ClusterOS,
	})
	Expect(err).NotTo(HaveOccurred())

	_, err = Td.CreateServiceAccount(clientNamespace, &clientSvcAccDef)
	Expect(err).NotTo(HaveOccurred())
	clientPod, err := Td.CreatePod(clientNamespace, clientPodDef)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreateService(clientNamespace, clientSvcDef)
	Expect(err).NotTo(HaveOccurred())

	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(clientNamespace, 60*time.Second, 1, nil)).To(Succeed())

	// Deploy allow rule client->server
	httpRG, trafficTarget := Td.CreateSimpleAllowPolicy(
		SimpleAllowPolicy{
			RouteGroupName:    "routes",
			TrafficTargetName: "target",

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

	return clientPod, serverPod, serverSvc
}

func verifySuccessfulPodConnection(srcPod, dstPod *v1.Pod, serverSvc *v1.Service) {
	By("Waiting for repeated request success")
	cond := Td.WaitForRepeatedSuccess(func() bool {
		result :=
			Td.FortioHTTPLoadTest(FortioHTTPLoadTestDef{
				HTTPRequestDef: HTTPRequestDef{
					SourceNs:        srcPod.Namespace,
					SourcePod:       srcPod.Name,
					SourceContainer: srcPod.Name,

					Destination: fmt.Sprintf("%s.%s:%d", serverSvc.Name, dstPod.Namespace, fortioHTTPPort),
				},
			})

		if result.Err != nil || result.HasFailedHTTPRequests() {
			Td.T.Logf("> REST req has failed requests: %v", result.Err)
			return false
		}
		Td.T.Logf("> REST req succeeded. Status codes: %v", result.AllReturnCodes())
		return true
	}, 2 /*runs the load test this many times successfully*/, 90*time.Second /*timeout*/)
	Expect(cond).To(BeTrue())
}

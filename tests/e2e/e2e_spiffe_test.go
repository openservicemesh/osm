package e2e

import (
	"fmt"
	"path/filepath"
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
	var testPod = framework.RandomNameWithPrefix("testpod")
	var meshNs = []string{testPod}

	It("has a Pod Certificate with a SPIFFE ID", func() {
		By("Installing OSM with SPIFFE Enabled")
		installOpts := Td.GetOSMInstallOpts(WithSpiffeEnabled())
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		// Create test NS in mesh
		By("Creating Namespace for test")
		for _, n := range meshNs {
			Expect(Td.CreateNs(n, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, n)).To(Succeed())
		}

		// Get simple pod definitions for the HTTP server
		svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
			SimplePodAppDef{
				PodName:   testPod,
				Namespace: testPod,
				Image:     "kennethreitz/httpbin",
				Ports:     []int{80},
				OS:        Td.ClusterOS,
			})
		Expect(err).NotTo(HaveOccurred())

		By("Creating a pod with Service")
		_, err = Td.CreateServiceAccount(testPod, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		pod, err := Td.CreatePod(testPod, podDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateService(testPod, svcDef)
		Expect(err).NotTo(HaveOccurred())

		Expect(Td.WaitForPodsRunningReady(testPod, 90*time.Second, 1, nil)).To(Succeed())

		verifySpiffeIDInPodCert(pod)
	})
}

func verifySpiffeIDInPodCert(pod *v1.Pod) {
	By("Verifying pod has SPIFFE cert in URI SAN")

	// It can take a moment for envoy to load the certs
	Eventually(func() (string, error) {
		args := []string{"proxy", "get", "certs", pod.Name, fmt.Sprintf("-n=%s", pod.Namespace)}
		stdout, _, err := Td.RunLocal(filepath.FromSlash("../../bin/osm"), args...)
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

package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	. "github.com/openservicemesh/osm/tests/framework"
)

// Prior iterations of OSM supported a wide range of min and max MTLS versions for the envoy sidecar (TLS_AUTO, TLSv1_0, TLSv1_1, TLSv1_2 and TLSv1_3)
// even though the OSM Control Plane's minimum version has been upgraded to TLSv1_2
// This test verifies that the envoy sidecar maxTLSVersion is compatible with the current OSM control plane's minTLSVersion
var _ = OSMDescribe("Test envoy maxTLSVersion is compatible with OSM control plane's minTLSVersion",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 12,
	},
	func() {
		Context("Envoy maxTLSVersion equals control planes's minTLSVersion, tls.VersionTLS12", func() {
			// Is compatible
			envoyMaxTLSVersion := "TLSv1_2"
			testEnvoyMaxMtlsVersionIsCompatibileWithOSMControlPlane(envoyMaxTLSVersion)
		})

		Context("envoy maxTLSVersion is greater than the control planes's minTLSVersion, tls.VersionTLS13", func() {
			// Is compatible
			envoyMaxTLSVersion := "TLSv1_3"
			testEnvoyMaxMtlsVersionIsCompatibileWithOSMControlPlane(envoyMaxTLSVersion)
		})

		Context("envoy maxTLSVersion is less than the control planes's minTLSVersion, tls.VersionTLS11", func() {
			// Is not compatible
			envoyMaxTLSVersion := "TLSv1_1"
			testEnvoyMaxMtlsVersionIsNotCompatibileWithOSMControlPlane(envoyMaxTLSVersion)
		})
	})

func testEnvoyMaxMtlsVersionIsCompatibileWithOSMControlPlane(envoyMaxTLSVersion string) {
	const clientName = "client"
	const serverName = "server"
	var ns = []string{clientName, serverName}

	It("Tests HTTP traffic for client pod -> server pod", func() {
		// Set up meshed client and server pods
		clientPod, dstSvc := setUpTestApps(envoyMaxTLSVersion, clientName, serverName, ns)

		By("Sending a request from client to server")
		// All ready. Expect client to reach server
		clientToServer := HTTPRequestDef{
			SourceNs:        clientName,
			SourcePod:       clientPod.Name,
			SourceContainer: clientName,

			Destination: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
		}

		srcToDestStr := fmt.Sprintf("%s -> %s",
			fmt.Sprintf("%s/%s", clientName, clientPod.Name),
			clientToServer.Destination)

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)
			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) HTTP Req failed %d %v",
					srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req succeeded: %d", srcToDestStr, result.StatusCode)
			return true
		}, 5, Td.ReqSuccessTimeout)
		Expect(cond).To(BeTrue(), "envoy maxTLSVersion %s is compatible with OSM control plane", envoyMaxTLSVersion)
	})
}

func testEnvoyMaxMtlsVersionIsNotCompatibileWithOSMControlPlane(envoyMaxTLSVersion string) {
	const clientName = "client"
	const serverName = "server"
	var ns = []string{clientName, serverName}

	It("Tests HTTP traffic for client pod -> server pod", func() {
		// Set up meshed client and server pods
		clientPod, dstSvc := setUpTestApps(envoyMaxTLSVersion, clientName, serverName, ns)

		By("Sending a request from client to server")
		// All ready. Expect client to reach server
		clientToServer := HTTPRequestDef{
			SourceNs:        clientName,
			SourcePod:       clientPod.Name,
			SourceContainer: clientName,

			Destination: fmt.Sprintf("%s.%s.svc.cluster.local", dstSvc.Name, dstSvc.Namespace),
		}

		srcToDestStr := fmt.Sprintf("%s -> %s",
			fmt.Sprintf("%s/%s", clientName, clientPod.Name),
			clientToServer.Destination)

		cond := Td.WaitForRepeatedSuccess(func() bool {
			result := Td.HTTPRequest(clientToServer)
			// Curl exit code 7 == Conn refused
			if result.Err == nil || !strings.Contains(result.Err.Error(), "command terminated with exit code 7 ") {
				Td.T.Logf("> (%s) HTTP Req failed, incorrect expected result: %d, %v", srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) HTTP Req failed correctly: %v", srcToDestStr, result.Err)
			return true
		}, 5, 150*time.Second)
		Expect(cond).To(BeTrue(), "envoy maxTLSVersion %s is not compatible with OSM control plane", envoyMaxTLSVersion)
	})
}

// setUpTestApps creates a curl client pod, http server pod and kubernetes service for server pod
func setUpTestApps(envoyMaxTLSVersion string, clientName string, serverName string, ns []string) (*v1.Pod, *v1.Service) {
	// Install OSM
	installOpts := Td.GetOSMInstallOpts()
	installOpts.EnablePermissiveMode = true
	Expect(Td.InstallOSM(installOpts)).To(Succeed())
	Expect(Td.WaitForPodsRunningReady(Td.OsmNamespace, 60*time.Second, 3 /* 1 controller, 1 injector, 1 bootstrap */, nil)).To(Succeed())

	// Get the meshConfig CRD
	meshConfig, err := Td.GetMeshConfig(Td.OsmNamespace)
	Expect(err).NotTo(HaveOccurred())

	// Update envoy maxTLSVersion
	By(fmt.Sprintf("Patching envoy maxTLSVersion to be %s", envoyMaxTLSVersion))
	meshConfig.Spec.Sidecar.TLSMaxProtocolVersion = envoyMaxTLSVersion
	_, err = Td.UpdateOSMConfig(meshConfig)
	Expect(err).NotTo(HaveOccurred())

	// Create Meshed Test NS
	for _, n := range ns {
		Expect(Td.CreateNs(n, nil)).To(Succeed())
		Expect(Td.AddNsToMesh(true, n)).To(Succeed())
	}

	// Get simple pod definitions for the HTTP server
	svcAccDef, podDef, svcDef, err := Td.GetOSSpecificHTTPBinPod(serverName, serverName, PodCommandDefault...)
	Expect(err).NotTo(HaveOccurred())

	// Create Server Pod
	_, err = Td.CreateServiceAccount(serverName, &svcAccDef)
	Expect(err).NotTo(HaveOccurred())
	_, err = Td.CreatePod(serverName, podDef)
	Expect(err).NotTo(HaveOccurred())

	// Create Server Service
	dstSvc, err := Td.CreateService(serverName, svcDef)
	Expect(err).NotTo(HaveOccurred())
	// Expect it to be up and running in it's receiver namespace
	Expect(Td.WaitForPodsRunningReady(serverName, 90*time.Second, 1, nil)).To(Succeed())

	// Create Client Pod
	withSourceKubernetesService := true
	// setupSource sets up a curl source service and returns the pod object
	clientPod := setupSource(clientName, withSourceKubernetesService)
	return clientPod, dstSvc
}

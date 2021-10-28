package e2e

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

const (
	httpRespHeaderName = "podname"
)

var _ = OSMDescribe("Test HTTP from N Clients deployments to 1 Server deployment backed with Traffic split test",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 5,
	},
	func() {
		Context("HTTP traffic splitting - root service selector matches backends", func() {
			// This test verifies traffic splitting where the root service selector matches
			// the pods backing the leaf services.
			// Below, all services and pods use the same app label, including the root service
			//
			// Split configuration:
			// root: httpbin-root labels(app: httpbin)
			// backends:
			//   - httpbin-v1 (weight: 50) labels(app: httpbin, version: v1)
			//   - httpbin-v2 (weight: 50) labels(app: httpbin, version: v2)
			//
			// Pods:
			// httpbin-v1 labels(app: httpbin, version: v1)
			// httpbin-v2 labels(app: httpbin, version: v2)
			//
			testTrafficSplitSelector()
		})
	})

func testTrafficSplitSelector() {
	It("Tests HTTP traffic split when root service selector matches backends", func() {
		if Td.DeployOnOpenShift {
			Skip("Skipping test: TrafficSplit selector test not supported on OpenShift")
		}

		clientNs := "client"
		clientName := "client"
		serviceNs := "server"
		serviceName := "httpbin"
		rootServiceName := "httpbin-root"
		appNamespaces := []string{clientNs, serviceNs}

		// Install OSM. Use permissive mode for simplicity
		Expect(Td.InstallOSM(Td.GetOSMInstallOpts())).To(Succeed())
		meshConfig, _ := Td.GetMeshConfig(Td.OsmNamespace)
		meshConfig.Spec.Traffic.EnablePermissiveTrafficPolicyMode = true
		_, err := Td.UpdateOSMConfig(meshConfig)
		Expect(err).NotTo(HaveOccurred())

		// Onboard namespaces
		Expect(Td.CreateMultipleNs(appNamespaces...)).To(Succeed())
		Expect(Td.AddNsToMesh(true, appNamespaces...)).To(Succeed())

		// Create deployments for the backend apps
		numBackends := 2
		for i := 1; i <= numBackends; i++ {
			Expect(deployBackendApp(fmt.Sprintf("v%d", i), serviceName, serviceNs)).To(Succeed())
		}
		Expect(Td.WaitForPodsRunningReady(serviceNs, 90*time.Second, 2, nil)).To(Succeed())

		// Create a root apex service with a selector that matches the backend
		Expect(createRootService(rootServiceName, serviceNs)).To(Succeed())

		// Create client app
		Expect(createClientApp(clientName, clientNs)).To(Succeed())
		Expect(Td.WaitForPodsRunningReady(clientNs, 90*time.Second, 1, nil)).To(Succeed())

		// Create SMI Traffic Split
		By("Creating TrafficSplit with root service selector matching backend pods")
		split := smiSplit.TrafficSplit{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-root",
				Namespace: serviceNs,
			},
			Spec: smiSplit.TrafficSplitSpec{
				Service: rootServiceName,
				Backends: []smiSplit.TrafficSplitBackend{
					{
						Service: fmt.Sprintf("%s-v1", serviceName),
						Weight:  50,
					},
					{
						Service: fmt.Sprintf("%s-v2", serviceName),
						Weight:  50,
					},
				},
			},
		}
		_, err = Td.CreateTrafficSplit(split.Namespace, split)
		Expect(err).To(BeNil())

		// Send traffic from client to root service and verify backends respond
		req := HTTPRequestDef{
			SourceNs:        clientNs,
			SourcePod:       clientName,
			SourceContainer: clientName,

			Destination: fmt.Sprintf("%s.%s:%d", rootServiceName, serviceNs, DefaultUpstreamServicePort),
		}

		srcToDestStr := fmt.Sprintf("%s/%s -> %s", clientNs, clientName, req.Destination)
		httpRespHeadersSeen := make(map[string]bool)
		cond := Td.WaitForSuccessAfterInitialFailure(func() bool {
			result := Td.HTTPRequest(req)

			if result.Err != nil || result.StatusCode != 200 {
				Td.T.Logf("> (%s) REST req failed (status: %d) %v", srcToDestStr, result.StatusCode, result.Err)
				return false
			}
			Td.T.Logf("> (%s) REST req succeeded: %d", srcToDestStr, result.StatusCode)
			dstPod, ok := result.Headers[httpRespHeaderName]
			if ok {
				// Store and mark that we have seen a response for this server pod
				httpRespHeadersSeen[dstPod] = true
			}
			return true
		}, 10 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
		Expect(cond).To(BeTrue())

		// Since there are 2 backends for the split config, we expect responses from both the backends
		numUniqueServerResp := 0
		for _, v := range httpRespHeadersSeen {
			if v {
				numUniqueServerResp++
			}
		}
		Expect(numUniqueServerResp).To(Equal(2))
	})
}

func deployBackendApp(version string, serviceName string, namespace string) error {
	appName := fmt.Sprintf("%s-%s", serviceName, version)
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.AppLabel: serviceName,
					"version":          version,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						constants.AppLabel: serviceName,
						"version":          version,
					},
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: new(int64), // 0
					NodeSelector: map[string]string{
						"kubernetes.io/os": Td.ClusterOS,
					},
					Containers: []corev1.Container{
						{
							Name:            appName,
							Image:           "simonkowallik/httpbin",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command:         HttpbinCmd,
							Ports: []corev1.ContainerPort{
								{
									Name:          "httpbin",
									ContainerPort: DefaultUpstreamServicePort,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: fmt.Sprintf("XHTTPBIN_%s", httpRespHeaderName),
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if _, err := Td.CreateDeployment(namespace, deployment); err != nil {
		return err
	}

	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				constants.AppLabel: serviceName,
				"version":          version,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				constants.AppLabel: serviceName,
				"version":          version,
			},
			Ports: []corev1.ServicePort{
				{
					Name:        "httpbin",
					AppProtocol: pointer.StringPtr("http"),
					Port:        DefaultUpstreamServicePort,
				},
			},
		},
	}

	_, err := Td.CreateService(namespace, service)
	return err
}

func createRootService(name string, namespace string) error {
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				constants.AppLabel: name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				constants.AppLabel: name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:        "httpbin",
					AppProtocol: pointer.StringPtr("http"),
					Port:        DefaultUpstreamServicePort,
				},
			},
		},
	}

	_, err := Td.CreateService(namespace, service)
	return err
}

func createClientApp(name string, namespace string) error {
	svcAccDef, podDef, _, err := Td.SimplePodApp(SimplePodAppDef{
		PodName:   name,
		Namespace: namespace,
		Command:   []string{"/bin/bash", "-c", "--"},
		Args:      []string{"while true; do sleep 30; done;"},
		Image:     "songrgg/alpine-debug",
		Ports:     []int{80},
		OS:        Td.ClusterOS,
	})
	if err != nil {
		return err
	}

	_, err = Td.CreateServiceAccount(namespace, &svcAccDef)
	if err != nil {
		return err
	}

	_, err = Td.CreatePod(namespace, podDef)
	return err
}

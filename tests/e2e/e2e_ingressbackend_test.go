package e2e

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"

	"github.com/openservicemesh/osm/tests/framework"
	. "github.com/openservicemesh/osm/tests/framework"
)

const (
	serverPort = 80
)

var _ = OSMDescribe("Ingress using IngressBackend API",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 6,
	},
	func() {
		Context("HTTP ingress with IngressBackend", func() {
			testIngressBackend()
		})
	})

func testIngressBackend() {
	var destNs = framework.RandomNameWithPrefix("server")

	It("allows ingress traffic", func() {
		// Install OSM
		installOpts := Td.GetOSMInstallOpts()
		Expect(Td.InstallOSM(installOpts)).To(Succeed())

		Expect(Td.CreateNs(destNs, nil)).To(Succeed())
		Expect(Td.AddNsToMesh(true, destNs)).To(Succeed())

		// Get simple pod definitions for the HTTP server
		svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
			SimplePodAppDef{
				PodName:   "server",
				Namespace: destNs,
				Image:     "kennethreitz/httpbin",
				Ports:     []int{serverPort},
				OS:        Td.ClusterOS,
			})
		Expect(err).NotTo(HaveOccurred())

		_, err = Td.CreateServiceAccount(destNs, &svcAccDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreatePod(destNs, podDef)
		Expect(err).NotTo(HaveOccurred())
		_, err = Td.CreateService(destNs, svcDef)
		Expect(err).NotTo(HaveOccurred())

		// Expect it to be up and running in it's receiver namespace
		Expect(Td.WaitForPodsRunningReady(destNs, 60*time.Second, 1, nil)).To(Succeed())

		// Install nginx ingress controller
		ingressAddr, err := Td.InstallNginxIngress()
		Expect(err).ToNot((HaveOccurred()))

		ing := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: svcDef.Name,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: pointer.StringPtr("nginx"),
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									// Adding root path due to nginx ingress issue: https://github.com/kubernetes/ingress-nginx/issues/8518
									{
										Path:     "/",
										PathType: (*networkingv1.PathType)(pointer.StringPtr(string(networkingv1.PathTypeImplementationSpecific))),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: svcDef.Name,
												Port: networkingv1.ServiceBackendPort{
													Number: serverPort,
												},
											},
										},
									},
									{
										Path:     "/status/200",
										PathType: (*networkingv1.PathType)(pointer.StringPtr(string(networkingv1.PathTypeImplementationSpecific))),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: svcDef.Name,
												Port: networkingv1.ServiceBackendPort{
													Number: serverPort,
												},
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
		_, err = Td.Client.NetworkingV1().Ingresses(destNs).Create(context.Background(), ing, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Requests should fail when no IngressBackend resource exists
		url := "http://" + ingressAddr + "/status/200"
		Td.T.Log("Checking requests to", url, "should fail")
		cond := Td.WaitForRepeatedSuccess(func() bool {
			resp, err := http.Get(url) // #nosec G107: Potential HTTP request made with variable url
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			if err != nil || status != 502 {
				Td.T.Logf("> REST req failed unexpectedly (status: %d) %v", status, err)
				return false
			}
			Td.T.Logf("> REST req failed expectedly: %d", status)
			return true
		}, 5 /*consecutive success threshold*/, 120*time.Second /*timeout*/)
		Expect(cond).To(BeTrue())

		By("Creating an IngressBackend policy")
		// Source in the ingress backend must be added to the mesh for endpoint discovery
		Expect(Td.AddNsToMesh(false, NginxIngressSvc.Namespace)).To(Succeed())

		ingressBackend := &policyv1alpha1.IngressBackend{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "httpbin-http",
				Namespace: destNs,
			},
			Spec: policyv1alpha1.IngressBackendSpec{
				Backends: []policyv1alpha1.BackendSpec{
					{
						Name: svcDef.Name,
						Port: policyv1alpha1.PortSpec{
							Number:   serverPort,
							Protocol: "http",
						},
					},
				},
				Sources: []policyv1alpha1.IngressSourceSpec{
					{
						Kind:      "Service",
						Name:      NginxIngressSvc.Name,
						Namespace: NginxIngressSvc.Namespace,
					},
				},
			},
		}

		_, err = Td.PolicyClient.PolicyV1alpha1().IngressBackends(ingressBackend.Namespace).Create(context.TODO(), ingressBackend, metav1.CreateOptions{})
		Expect(err).ToNot((HaveOccurred()))

		// Expect client to reach server
		Td.T.Log("Checking requests to", url, "should succeed")
		cond = Td.WaitForRepeatedSuccess(func() bool {
			resp, err := http.Get(url) // #nosec G107: Potential HTTP request made with variable url
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			if err != nil || status != 200 {
				Td.T.Logf("> REST req failed (status: %d) %v", status, err)
				return false
			}
			Td.T.Logf("> REST req succeeded: %d", status)
			return true
		}, 5 /*consecutive success threshold*/, 120*time.Second /*timeout*/)
		Expect(cond).To(BeTrue())
	})
}

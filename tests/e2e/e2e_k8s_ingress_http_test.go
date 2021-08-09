package e2e

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("HTTP ingress using k8s Ingress API",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 5,
	},
	func() {
		const destNs = "server"

		It("allows HTTP ingress traffic", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			installOpts.SetOverrides = []string{"OpenServiceMesh.featureFlags.enableIngressBackendPolicy=false"}
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			Expect(Td.CreateNs(destNs, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, destNs)).To(Succeed())

			// Get simple pod definitions for the HTTP server
			svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
				SimplePodAppDef{
					Name:      "server",
					Namespace: destNs,
					Image:     "kennethreitz/httpbin",
					Ports:     []int{80},
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

			// Requests should fail when no ingress exists
			url := "http://" + ingressAddr + "/status/200"
			Td.T.Log("Checking requests to", url, "should fail")
			cond := Td.WaitForRepeatedSuccess(func() bool {
				resp, err := http.Get(url) // #nosec G107: Potential HTTP request made with variable url
				status := 0
				if resp != nil {
					status = resp.StatusCode
				}
				if err != nil || status != 404 {
					Td.T.Logf("> REST req failed unexpectedly (status: %d) %v", status, err)
					return false
				}
				Td.T.Logf("> REST req failed expectedly: %d", status)
				return true
			}, 5 /*consecutive success threshold*/, 120*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())

			ing := &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: svcDef.Name,
				},
				Spec: v1beta1.IngressSpec{
					IngressClassName: pointer.StringPtr("nginx"),
					Rules: []v1beta1.IngressRule{
						{
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										{
											Path:     "/status/200",
											PathType: (*v1beta1.PathType)(pointer.StringPtr(string(v1beta1.PathTypeImplementationSpecific))),
											Backend: v1beta1.IngressBackend{
												ServiceName: svcDef.Name,
												ServicePort: intstr.FromInt(80),
											},
										},
									},
								},
							},
						},
					},
				},
			}
			_, err = Td.Client.NetworkingV1beta1().Ingresses(destNs).Create(context.Background(), ing, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// All ready. Expect client to reach server
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
	})

package e2e

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("HTTP ingress",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 3,
	},
	func() {
		const destNs = "server"

		It("allows HTTP ingress traffic", func() {
			// Install OSM
			installOpts := Td.GetOSMInstallOpts()
			Expect(Td.InstallOSM(installOpts)).To(Succeed())

			Expect(Td.CreateNs(destNs, nil)).To(Succeed())
			Expect(Td.AddNsToMesh(true, destNs)).To(Succeed())

			// Get simple pod definitions for the HTTP server
			svcAccDef, podDef, svcDef := Td.SimplePodApp(
				SimplePodAppDef{
					Name:      "server",
					Namespace: destNs,
					Image:     "kennethreitz/httpbin",
					Ports:     []int{80},
				})

			_, err := Td.CreateServiceAccount(destNs, &svcAccDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreatePod(destNs, podDef)
			Expect(err).NotTo(HaveOccurred())
			_, err = Td.CreateService(destNs, svcDef)
			Expect(err).NotTo(HaveOccurred())

			// Expect it to be up and running in it's receiver namespace
			Expect(Td.WaitForPodsRunningReady(destNs, 60*time.Second, 1)).To(Succeed())

			// Install nginx ingress controller
			helm := &action.Configuration{}
			Expect(helm.Init(Td.Env.RESTClientGetter(), Td.OsmNamespace, "secret", Td.T.Logf)).To(Succeed())
			install := action.NewInstall(helm)
			install.RepoURL = "https://kubernetes.github.io/ingress-nginx"
			install.Namespace = Td.OsmNamespace
			install.ReleaseName = "ingress-nginx"
			install.Version = "3.23.0"
			install.Wait = true
			install.Timeout = 5 * time.Minute
			chartPath, err := install.LocateChart("ingress-nginx", helmcli.New())
			Expect(err).NotTo(HaveOccurred())
			chart, err := loader.Load(chartPath)
			Expect(err).NotTo(HaveOccurred())
			_, err = install.Run(chart, map[string]interface{}{
				"controller": map[string]interface{}{
					"hostPort": map[string]interface{}{
						"enabled": true,
					},
					"service": map[string]interface{}{
						"type": "NodePort",
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Requests should fail when no ingress exists
			cond := Td.WaitForRepeatedSuccess(func() bool {
				resp, err := http.Get("http://localhost/status/200")
				if err != nil || resp.StatusCode != 404 {
					Td.T.Logf("> REST req failed unexpectedly (status: %d) %v", resp.StatusCode, err)
					return false
				}
				Td.T.Logf("> REST req failed expectedly: %d", resp.StatusCode)
				return true
			}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())

			ing := &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name: svcDef.Name,
					Annotations: map[string]string{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						{
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										{
											Path: "/status/200",
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
			cond = Td.WaitForRepeatedSuccess(func() bool {
				resp, err := http.Get("http://localhost/status/200")
				if err != nil || resp.StatusCode != 200 {
					Td.T.Logf("> REST req failed (status: %d) %v", resp.StatusCode, err)
					return false
				}
				Td.T.Logf("> REST req succeeded: %d", resp.StatusCode)
				return true
			}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())
		})
	})

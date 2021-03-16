package e2e

import (
	"context"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmcli "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
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
			// Install Service as NodePort on kind, LoadBalancer elsewhere

			// Check the node's provider so this works for preprovisioned kind clusters
			nodes, err := Td.Client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			providerID := nodes.Items[0].Spec.ProviderID
			isKind := strings.HasPrefix(providerID, "kind://")
			var vals map[string]interface{}
			if isKind {
				vals = map[string]interface{}{
					"controller": map[string]interface{}{
						"hostPort": map[string]interface{}{
							"enabled": true,
						},
						"service": map[string]interface{}{
							"type": "NodePort",
						},
					},
				}
			}

			helm := &action.Configuration{}
			Expect(helm.Init(Td.Env.RESTClientGetter(), Td.OsmNamespace, "secret", Td.T.Logf)).To(Succeed())
			helm.KubeClient.(*kube.Client).Namespace = Td.OsmNamespace
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
			_, err = install.Run(chart, vals)
			Expect(err).NotTo(HaveOccurred())

			ingressAddr := "localhost"
			if !isKind {
				svc, err := Td.Client.CoreV1().Services(Td.OsmNamespace).Get(context.Background(), "ingress-nginx-controller", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				ingressAddr = svc.Status.LoadBalancer.Ingress[0].IP
				if len(ingressAddr) == 0 {
					ingressAddr = svc.Status.LoadBalancer.Ingress[0].Hostname
				}
			}

			// Requests should fail when no ingress exists
			url := "http://" + ingressAddr + "/status/200"
			Td.T.Log("Checking requests to", url, "should fail")
			cond := Td.WaitForRepeatedSuccess(func() bool {
				resp, err := http.Get(url)
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
			Td.T.Log("Checking requests to", url, "should succeed")
			cond = Td.WaitForRepeatedSuccess(func() bool {
				resp, err := http.Get(url)
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
			}, 5 /*consecutive success threshold*/, 60*time.Second /*timeout*/)
			Expect(cond).To(BeTrue())
		})
	})

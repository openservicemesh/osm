package main

import (
	"bytes"
	"context"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
)

var _ = FDescribe("Running the dashboard command", func() {

	Describe("with default parameters", func() {
		var (
			out           *bytes.Buffer
			config        *helm.Configuration
			err           error
			fakeClientSet kubernetes.Interface
		)

		BeforeEach(func() {

			out = new(bytes.Buffer)
			config = &helm.Configuration{
				//Releases: store,
				KubeClient: &kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
				Capabilities: chartutil.DefaultCapabilities,
				Log:          func(format string, v ...interface{}) {},
			}

			fakeClientSet = fake.NewSimpleClientset()

			serviceSpec := &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: grafanaServiceName,
				},
				Spec: v1.ServiceSpec{
					Selector: map[string]string{
						"app": "grafana",
					},
				},
			}

			podSpec := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "grafana",
					},
				},
				Status: v1.PodStatus{
					Phase: "Running",
				},
			}

			/*pods, err := v1ClientSet.Pods(settings.Namespace()).
			List(context.TODO(), listOptions)

			// Will select first running Pod available
			it := 0
			for {
				if pods.Items[it].Status.Phase == "Running" {
					break
				}

				it++
				if it == len(pods.Items) {
					log.Fatalf("No running Grafana pod available.")
				}
			} */

			fakeClientSet.CoreV1().Services(settings.Namespace()).Create(context.TODO(), serviceSpec, metav1.CreateOptions{})
			fakeClientSet.CoreV1().Pods(settings.Namespace()).Create(context.TODO(), podSpec, metav1.CreateOptions{})

			dashboardCmd := &dashboardCmd{
				out:         out,
				localPort:   grafanaWebPort,
				remotePort:  grafanaWebPort,
				openBrowser: true,
				config:      config,
				clientSet:   fakeClientSet,
			}

			err = dashboardCmd.run()
		})

		It("should not error", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		/*It("should give a message confirming the successful install", func() {
			Expect(out.String()).To(Equal("OSM installed successfully in namespace [osm-system] with mesh name [osm]\n"))
		}) */

		/* Context("the Helm release", func() {
			var (
				rel *release.Release
				err error
			)

			BeforeEach(func() {
				rel, err = config.Releases.Get(defaultMeshName, 1)
			})

			It("should not error when retrieved", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have the correct values", func() {
				Expect(rel.Config).To(BeEquivalentTo(map[string]interface{}{
					"OpenServiceMesh": map[string]interface{}{
						"localPort": 	grafanaWebPort,
						"remotePort":	grafanaWebPort,
						"openBrowser":	true,
					}}))
			})

			It("should be installed in the correct namespace", func() {
				Expect(rel.Namespace).To(Equal(settings.Namespace()))
			})
		}) */

	})

})

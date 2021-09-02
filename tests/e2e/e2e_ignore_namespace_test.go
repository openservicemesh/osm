package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Ignore Namespaces",
	OSMDescribeInfo{
		Tier:   1,
		Bucket: 2,
	},
	func() {
		Context("Ignore Label", func() {
			const ignoreNs = "ignore"
			const monitorIgnoreNs = "mesh-ignore"
			const sidecarMonitorIgnoreNs = "sidecar-monitor-ignore"
			var ns []string = []string{ignoreNs, monitorIgnoreNs, sidecarMonitorIgnoreNs}

			It("Tests the ignore label on a namespace disables sidecar injection", func() {
				// Install OSM
				installOpts := Td.GetOSMInstallOpts()
				installOpts.EnablePermissiveMode = true
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				// Create test NS in mesh with ignore label
				for _, n := range ns {
					Expect(Td.CreateNs(n, map[string]string{constants.IgnoreLabel: "true"})).To(Succeed())
				}

				// Add monitor-ignore to mesh with sidecar injection disabled
				Expect(Td.AddNsToMesh(false, monitorIgnoreNs)).To(Succeed())
				// Add sidecar-monitor-ignore to mesh with sidecar injection enabled
				Expect(Td.AddNsToMesh(true, sidecarMonitorIgnoreNs)).To(Succeed())

				By("Ensuring pod is not injected with a sidecar when added a namespace with the ignore label")

				svcAccDef, podDef, svcDef, err := Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "pod1",
						Namespace: ignoreNs,
						Command:   []string{"/bin/bash", "-c", "--"},
						Args:      []string{"while true; do sleep 30; done;"},
						Image:     "songrgg/alpine-debug",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(ignoreNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				pod, err := Td.CreatePod(ignoreNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(ignoreNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(ignoreNs, 90*time.Second, 1, nil)).To(Succeed())

				pod, err = Td.Client.CoreV1().Pods(ignoreNs).Get(context.Background(), pod.Name, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(hasSidecar(pod.Spec.Containers)).To(BeFalse())

				By("Ensuring pod with a sidecar injection label is not injected with a sidecar when added to a namespace with the ignore label")

				svcAccDef, podDef, svcDef, err = Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "pod2",
						Namespace: ignoreNs,
						Command:   []string{"/bin/bash", "-c", "--"},
						Args:      []string{"while true; do sleep 30; done;"},
						Image:     "songrgg/alpine-debug",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())
				podDef.Annotations = map[string]string{constants.SidecarInjectionAnnotation: "enabled"}

				_, err = Td.CreateServiceAccount(ignoreNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				pod, err = Td.CreatePod(ignoreNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(ignoreNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(ignoreNs, 90*time.Second, 1, nil)).To(Succeed())

				pod, err = Td.Client.CoreV1().Pods(ignoreNs).Get(context.Background(), pod.Name, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(hasSidecar(pod.Spec.Containers)).To(BeFalse())

				By("Ensuring a pod is not injected with a sidecar when added to namespace with the monitored-by and ignore labels")

				svcAccDef, podDef, svcDef, err = Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "pod1",
						Namespace: monitorIgnoreNs,
						Command:   []string{"/bin/bash", "-c", "--"},
						Args:      []string{"while true; do sleep 30; done;"},
						Image:     "songrgg/alpine-debug",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(monitorIgnoreNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				pod, err = Td.CreatePod(monitorIgnoreNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(monitorIgnoreNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(monitorIgnoreNs, 90*time.Second, 1, nil)).To(Succeed())

				pod, err = Td.Client.CoreV1().Pods(monitorIgnoreNs).Get(context.Background(), pod.Name, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(hasSidecar(pod.Spec.Containers)).To(BeFalse())

				By("Ensuring pod with a sidecar injection label is not injected with a sidecar when added to a namespace with the monitored-by and ignore labels")

				svcAccDef, podDef, svcDef, err = Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "pod2",
						Namespace: monitorIgnoreNs,
						Command:   []string{"/bin/bash", "-c", "--"},
						Args:      []string{"while true; do sleep 30; done;"},
						Image:     "songrgg/alpine-debug",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())
				podDef.Annotations = map[string]string{constants.SidecarInjectionAnnotation: "enabled"}

				_, err = Td.CreateServiceAccount(monitorIgnoreNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				pod, err = Td.CreatePod(monitorIgnoreNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(monitorIgnoreNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(monitorIgnoreNs, 90*time.Second, 1, nil)).To(Succeed())

				pod, err = Td.Client.CoreV1().Pods(monitorIgnoreNs).Get(context.Background(), pod.Name, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(hasSidecar(pod.Spec.Containers)).To(BeFalse())

				By("Ensuring a pod is not injected with a sidecar when added to namespace with the monitored-by, ignore, and sidecar injection labels set")

				// Get simple Pod definitions
				svcAccDef, podDef, svcDef, err = Td.SimplePodApp(
					SimplePodAppDef{
						Name:      "pod1",
						Namespace: sidecarMonitorIgnoreNs,
						Command:   []string{"/bin/bash", "-c", "--"},
						Args:      []string{"while true; do sleep 30; done;"},
						Image:     "songrgg/alpine-debug",
						Ports:     []int{80},
						OS:        Td.ClusterOS,
					})
				Expect(err).NotTo(HaveOccurred())

				_, err = Td.CreateServiceAccount(sidecarMonitorIgnoreNs, &svcAccDef)
				Expect(err).NotTo(HaveOccurred())
				pod, err = Td.CreatePod(sidecarMonitorIgnoreNs, podDef)
				Expect(err).NotTo(HaveOccurred())
				_, err = Td.CreateService(sidecarMonitorIgnoreNs, svcDef)
				Expect(err).NotTo(HaveOccurred())

				Expect(Td.WaitForPodsRunningReady(sidecarMonitorIgnoreNs, 90*time.Second, 1, nil)).To(Succeed())

				pod, err = Td.Client.CoreV1().Pods(sidecarMonitorIgnoreNs).Get(context.Background(), pod.Name, v1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(hasSidecar(pod.Spec.Containers)).To(BeFalse())
			})
		})
	})

func hasSidecar(containers []corev1.Container) bool {
	for _, container := range containers {
		if container.Name == constants.EnvoyContainerName {
			return true
		}
	}
	return false
}

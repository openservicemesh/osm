package e2e

import (
	"bytes"
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openservicemesh/osm/tests/framework"
)

var _ = OSMDescribe("Test deployment of Fluent Bit sidecar",
	OSMDescribeInfo{
		Tier:   2,
		Bucket: 2,
	},
	func() {
		Context("Fluent Bit output", func() {
			It("Forwards correctly filtered logs to stdout", func() {
				// Install OSM with Fluentbit
				installOpts := Td.GetOSMInstallOpts()
				installOpts.DeployFluentbit = true
				installOpts.SetOverrides = []string{"OpenServiceMesh.fluentBit.logLevel=info"}
				Expect(Td.InstallOSM(installOpts)).To(Succeed())

				pods, err := Td.Client.CoreV1().Pods(Td.OsmNamespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{"app": "osm-controller"}).String(),
				})
				Expect(err).NotTo(HaveOccurred())

				// Query fluentbit-logger container logs to test if Fluent Bit filters are working
				fluentBitContainerFound := false
				for _, pod := range pods.Items {
					// Wait until osm-controller has generated a log to check against
					logLevel := "\"level\":\"info\""
					err := waitForLogEmission(pod.Namespace, pod.Name, "osm-controller", logLevel)
					Expect(err).To(BeNil())
					for _, container := range pod.Spec.Containers {
						if strings.Contains(container.Image, "fluent-bit") {
							fluentBitContainerFound = true
							podLogs, err := getPodLogs(pod.Namespace, pod.Name, "fluentbit-logger")
							Expect(err).ToNot(HaveOccurred(), "Unable to get container logs")
							Expect(podLogs).To(ContainSubstring(logLevel))
							Expect(podLogs).To(ContainSubstring("\"controller_pod_name\"=>\"osm-controller-"))
							Expect(podLogs).NotTo(ContainSubstring("\"keep\"=>\"true\""))
						}
					}
				}
				Expect(fluentBitContainerFound).To(BeTrue())
			})
		})
	})

func waitForLogEmission(namespace, podName, containerName, logString string) error {
	return wait.Poll(time.Second*3, time.Second*180, isLogEmitted(namespace, podName, containerName, logString))
}

// Checks whether expected string has been logged yet
func isLogEmitted(namespace, podName, containerName, expectedLog string) wait.ConditionFunc {
	return func() (bool, error) {
		podLogs, err := getPodLogs(namespace, podName, containerName)
		if err != nil {
			return false, err
		}
		if strings.Contains(podLogs, expectedLog) {
			return true, nil
		}
		return false, nil
	}
}

// getPodLogs returns pod logs
func getPodLogs(namespace string, podName string, containerName string) (string, error) {
	podLogOptions := &corev1.PodLogOptions{
		Container: containerName,
		Follow:    false,
	}

	logStream, err := Td.Client.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions).Stream(context.TODO())
	if err != nil {
		return "Error in opening stream", err
	}

	defer logStream.Close() //nolint: errcheck,gosec
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(logStream)
	if err != nil {
		return "Error reading from pod logs stream", err
	}
	return buf.String(), nil
}

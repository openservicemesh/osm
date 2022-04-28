package bugreport

import (
	"context"
	"fmt"
	"path"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func (c *Config) collectControlPlaneLogs() {
	pods, err := c.KubeClient.CoreV1().Pods(c.ControlPlaneNamepace).
		List(context.Background(), v1.ListOptions{
			LabelSelector: fmt.Sprintf(
				"%s=%s",
				constants.OSMAppNameLabelKey,
				constants.OSMAppNameLabelValue,
			),
		})
	if err != nil {
		c.completionFailure("Error getting control plane pods: %s", err)
		goto afterPodLogCollection
	}

	for _, pod := range pods.Items {
		cmd := []string{"kubectl", "logs", "-n", pod.Namespace, pod.Name}
		outPath := path.Join(c.rootNamespaceDirPath(), pod.Namespace, rootPodDirName, pod.Name, commandsDirName, strings.Join(cmd, "_"))
		if err := runCmdAndWriteToFile(cmd, outPath); err != nil {
			c.completionFailure("Error writing control pod logs: %s", err)
			continue
		}
		c.completionSuccess("Collected report for Pod %s/%s", pod.Namespace, pod.Name)

		if pod.Status.ContainerStatuses[0].RestartCount > 0 {
			cmd = append(cmd, "--previous")
			outPath = path.Join(c.rootNamespaceDirPath(), pod.Namespace, rootPodDirName, pod.Name, commandsDirName, strings.Join(cmd, "_"))
			if err := runCmdAndWriteToFile(cmd, outPath); err != nil {
				c.completionFailure("Error writing previous control pod logs: %s", err)
				continue
			}
			c.completionSuccess("Collected previous report for Pod %s/%s", pod.Namespace, pod.Name)
		}
	}

afterPodLogCollection:
	// Collect MeshConfig, secrets in control plane namespace
	namespaceCmds := [][]string{
		{"kubectl", "get", "meshconfig", "-n", c.ControlPlaneNamepace, "-o", "yaml"},
		{"kubectl", "get", "secrets", "-n", c.ControlPlaneNamepace}, // Only collect names to avoid collecting sensitive info
	}
	for _, cmd := range namespaceCmds {
		outPath := path.Join(c.rootNamespaceDirPath(), c.ControlPlaneNamepace, commandsDirName, strings.Join(cmd, "_"))
		if err := runCmdAndWriteToFile(cmd, outPath); err != nil {
			c.completionFailure("Error writing control namespace scoped report: %s", err)
			continue
		}
	}
}

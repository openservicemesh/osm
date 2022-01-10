package bugreport

import (
	"context"
	"fmt"
	"path"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/constants"
)

func (c *Config) collectControlPlaneLogs() error {
	pods, err := c.KubeClient.CoreV1().Pods(c.ControlPlaneNamepace).
		List(context.Background(), v1.ListOptions{
			LabelSelector: fmt.Sprintf(
				"%s=%s",
				constants.OSMAppNameLabelKey,
				constants.OSMAppNameLabelValue,
			),
		})
	if err != nil {
		return fmt.Errorf("error getting control plane pods: %w", err)
	}

	for _, pod := range pods.Items {
		cmd := []string{"kubectl", "logs", "-n", pod.Namespace, pod.Name}
		outPath := path.Join(c.rootNamespaceDirPath(), pod.Namespace, rootPodDirName, pod.Name, commandsDirName, strings.Join(cmd, "_"))
		if err := runCmdAndWriteToFile(cmd, outPath); err != nil {
			return fmt.Errorf("error writing control pod logs: %w", err)
		}
		c.completionSuccess("Collected report for Pod %s/%s", pod.Namespace, pod.Name)

		if pod.Status.ContainerStatuses[0].RestartCount > 0 {
			cmd = append(cmd, "--previous")
			outPath = path.Join(c.rootNamespaceDirPath(), pod.Namespace, rootPodDirName, pod.Name, commandsDirName, strings.Join(cmd, "_"))
			if err := runCmdAndWriteToFile(cmd, outPath); err != nil {
				return fmt.Errorf("error writing previous control pod logs: %w", err)
			}
			c.completionSuccess("Collected previous report for Pod %s/%s", pod.Namespace, pod.Name)
		}
	}

	return nil
}

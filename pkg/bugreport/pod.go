package bugreport

import (
	"context"
	"path"
	"strings"

	mapset "github.com/deckarep/golang-set"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
)

const (
	rootPodDirName = "pods"
)

func (c *Config) collectPerPodReport() {
	for _, pod := range c.AppPods {
		for _, podCmd := range getPerPodCommands(pod) {
			outPath := path.Join(c.rootNamespaceDirPath(), pod.Namespace, rootPodDirName, pod.Name, commandsDirName, strings.Join(podCmd, "_"))
			if err := runCmdAndWriteToFile(podCmd, outPath); err != nil {
				c.completionFailure("Error running cmd: %v", podCmd)
			}
		}
		c.completionSuccess("Collected report for Pod %q", pod)
	}
}

func (c *Config) getUniquePods() []types.NamespacedName {
	var pods []types.NamespacedName
	podSet := mapset.NewSet()

	for _, pod := range c.AppPods {
		podSet.Add(pod)
	}

	for _, deployment := range c.AppDeployments {
		d, err := c.KubeClient.AppsV1().Deployments(deployment.Namespace).Get(context.Background(), deployment.Name, metav1.GetOptions{})
		if err != nil {
			c.completionFailure("Deployment %s not found, skipping it", deployment)
			continue
		}
		podLabels := d.Spec.Template.ObjectMeta.Labels
		options := metav1.ListOptions{
			LabelSelector: fields.SelectorFromSet(podLabels).String(),
		}
		podList, err := c.KubeClient.CoreV1().Pods(deployment.Namespace).List(context.Background(), options)
		if err != nil {
			c.completionFailure("Error listing Pods for Deployment %s, err: %s", deployment, err)
		}
		if podList == nil {
			continue
		}
		for _, pod := range podList.Items {
			p := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
			podSet.Add(p)
		}
	}

	for p := range podSet.Iter() {
		pods = append(pods, p.(types.NamespacedName))
	}

	return pods
}

func getPerPodCommands(pod types.NamespacedName) [][]string {
	return [][]string{
		{"kubectl", "logs", pod.Name, "-n", pod.Namespace, "-c", "envoy"},
		{"osm", "proxy", "get", "config_dump", pod.Name, "-n", pod.Namespace},
		{"osm", "proxy", "get", "ready", pod.Name, "-n", pod.Namespace},
		{"osm", "proxy", "get", "stats", pod.Name, "-n", pod.Namespace},
		{"osm", "proxy", "get", "clusters", pod.Name, "-n", pod.Namespace},
		{"osm", "proxy", "get", "listeners", pod.Name, "-n", pod.Namespace},
		{"osm", "proxy", "get", "certs", pod.Name, "-n", pod.Namespace},
	}
}

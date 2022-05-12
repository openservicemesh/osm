package bugreport

import (
	"context"
	"path"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nginxIngressControllerLabelSelector   string = "app.kubernetes.io/name=ingress-nginx"
	contourIngressControllerLabelSelector string = "app.kubernetes.io/name=contour"

	ingressResourceName        string = "ingresses"
	ingressBackendResourceName string = "ingressbackends"
)

func (c *Config) collectIngressReport() {
	namespaceList, err := c.KubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		c.completionFailure("Error listing namespaces: %s", err)
		return
	}

	for _, namespace := range namespaceList.Items {
		ingressList, err := c.KubeClient.NetworkingV1().Ingresses(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			c.completionFailure("Error fetching Ingresses in namespace %s: %s", namespace.Name, err)
			continue
		}

		for _, ingress := range ingressList.Items {
			for _, cmd := range ingressReportCommands(namespace.Name, ingress.Name, ingressResourceName) {
				filename := path.Join(
					c.rootNamespaceDirPath(),
					namespace.Name,
					ingressResourceName,
					ingress.Name,
					commandsDirName,
					strings.Join(cmd, "_"),
				)
				if err := runCmdAndWriteToFile(cmd, filename); err != nil {
					c.completionFailure("Error writing Ingress command: %s", err)
					continue
				}
			}
			c.completionSuccess("Collected report for ingress %s/%s", ingress.Namespace, ingress.Name)
		}

		ingressBackendList, err := c.PolicyClient.PolicyV1alpha1().IngressBackends(namespace.Name).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			c.completionFailure("Error fetching IngressBackends in namespace %s: %s", namespace.Name, err)
			continue
		}

		for _, ingressBackend := range ingressBackendList.Items {
			for _, cmd := range ingressReportCommands(namespace.Name, ingressBackend.Name, ingressBackendResourceName) {
				filename := path.Join(
					c.rootNamespaceDirPath(),
					namespace.Name,
					ingressBackendResourceName,
					ingressBackend.Name,
					commandsDirName,
					strings.Join(cmd, "_"),
				)
				if err := runCmdAndWriteToFile(cmd, filename); err != nil {
					c.completionFailure("Error writing IngressBackend command: %s", err)
					continue
				}
			}
			c.completionSuccess("Collected report for ingress backend %s/%s", ingressBackend.Namespace, ingressBackend.Name)
		}
	}
}

func (c *Config) collectIngressControllerReport() {
	ingressControllerLabelsSelectors := []string{
		nginxIngressControllerLabelSelector,
		contourIngressControllerLabelSelector,
	}
	for _, labelSelector := range ingressControllerLabelsSelectors {
		if err := c.collectIngressControllerReportByLabelSelector(labelSelector); err != nil {
			c.completionFailure("Error generating ingress controller report: %s", err)
		}
	}
}

func (c *Config) collectIngressControllerReportByLabelSelector(labelSelector string) error {
	namespaceList, err := c.KubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, namespace := range namespaceList.Items {
		podList, err := c.KubeClient.CoreV1().Pods(namespace.Name).List(
			context.Background(),
			metav1.ListOptions{LabelSelector: labelSelector},
		)
		if err != nil {
			c.completionFailure("Error fetching ingress pods in namespace %s: %s", namespace.Name, err)
			continue
		}

		for _, pod := range podList.Items {
			cmds := append(
				ingressControllerReportCommands(pod.Namespace, pod.Name),
				ingressReportCommands(pod.Namespace, pod.Name, "pods")...,
			)
			for _, cmd := range cmds {
				filename := path.Join(
					c.rootNamespaceDirPath(),
					namespace.Name,
					rootPodDirName,
					pod.Name,
					commandsDirName,
					strings.Join(cmd, "_"),
				)
				if err := runCmdAndWriteToFile(cmd, filename); err != nil {
					c.completionFailure("Error writing ingress command: %s", err)
					continue
				}
			}
			c.completionSuccess("Collected report for ingress controller %s/%s", pod.Namespace, pod.Name)
		}
	}

	return nil
}

func ingressReportCommands(namespace, resourceName, resourceType string) [][]string {
	return [][]string{
		{"kubectl", "get", resourceType, "-n", namespace, resourceName, "-o", "yaml"},
		{"kubectl", "describe", resourceType, "-n", namespace, resourceName},
	}
}

func ingressControllerReportCommands(namespace, resourceName string) [][]string {
	return [][]string{
		{"kubectl", "logs", "-n", namespace, resourceName},
	}
}

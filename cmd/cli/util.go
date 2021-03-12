package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// confirm displays a prompt `s` to the user and returns a bool indicating yes / no
// If the lowercased, trimmed input begins with anything other than 'y', it returns false
// It accepts an int `tries` representing the number of attempts before returning false
func confirm(stdin io.Reader, stdout io.Writer, s string, tries int) (bool, error) {
	r := bufio.NewReader(stdin)

	for ; tries > 0; tries-- {
		fmt.Fprintf(stdout, "%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(res)) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			continue
		}
	}

	return false, nil
}

func getMeshes(clientSet kubernetes.Interface, meshName string, namespace string) ([]v1.Deployment, error) {
	deploymentsClient := clientSet.AppsV1().Deployments(namespace)
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"meshName": meshName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	osmControllerDeployments, err := deploymentsClient.List(context.TODO(), listOptions)
	return osmControllerDeployments.Items, err
}

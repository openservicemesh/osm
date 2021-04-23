package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

const meshListDescription = `
This command will list all the osm control planes running in a Kubernetes cluster and controller pods.`

type meshListCmd struct {
	out       io.Writer
	clientSet kubernetes.Interface
}

func newMeshList(out io.Writer) *cobra.Command {
	meshList := &meshListCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "list control planes in k8s cluster",
		Long:  meshListDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			meshList.clientSet = clientset
			return meshList.run()
		},
	}

	return cmd
}

func (l *meshListCmd) run() error {
	list, err := getControllerDeployments(l.clientSet)
	if err != nil {
		return errors.Errorf("Could not list deployments %v", err)
	}
	if len(list.Items) == 0 {
		fmt.Fprintf(l.out, "No control planes found\n")
		return nil
	}

	w := newTabWriter(l.out)

	fmt.Fprintln(w, "\nMESH NAME\tNAMESPACE\tCONTROLLER PODS\tVERSION")
	for _, elem := range list.Items {
		m := elem.ObjectMeta.Labels["meshName"]
		ns := elem.ObjectMeta.Namespace
		x := getNamespacePods(l.clientSet, m, ns)
		v := elem.ObjectMeta.Labels[constants.OSMAppVersionLabelKey]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m, ns, strings.Join(x["Pods"], ","), v)
	}
	_ = w.Flush()

	return nil
}

// getNamespacePods returns a map of controller pods
func getNamespacePods(clientSet kubernetes.Interface, m string, ns string) map[string][]string {
	x := make(map[string][]string)

	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	pods, _ := clientSet.CoreV1().Pods(ns).List(context.TODO(), listOptions)

	for pno := 0; pno < len(pods.Items); pno++ {
		x["Pods"] = append(x["Pods"], pods.Items[pno].GetName())
	}

	return x
}

// getControllerDeployments returns a list of Deployments corresponding to osm-controller
func getControllerDeployments(clientSet kubernetes.Interface) (*v1.DeploymentList, error) {
	deploymentsClient := clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

// getMeshNames returns a set of mesh names corresponding to meshes within the cluster
func getMeshNames(clientSet kubernetes.Interface) mapset.Set {
	meshList := mapset.NewSet()

	deploymentList, _ := getControllerDeployments(clientSet)
	for _, elem := range deploymentList.Items {
		meshList.Add(elem.ObjectMeta.Labels["meshName"])
	}

	return meshList
}

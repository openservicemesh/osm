package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

const meshListDescription = `
This command will list all the osm control planes running in a Kubernetes cluster and their namespaces.`

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
				return errors.Errorf("Error fetching kubeconfig")
			}
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			meshList.clientSet = clientset
			return meshList.run()
		},
	}

	return cmd
}

func (l *meshListCmd) run() error {
	list, err := l.selectMeshes()
	if err != nil {
		return errors.Errorf("Could not list deployments %v", err)
	}
	if len(list.Items) == 0 {
		fmt.Fprintf(l.out, "No control planes found\n")
		return nil
	}

	w := newTabWriter(l.out)
	fmt.Fprintln(w, "MESH NAME\tNAMESPACE\t")
	for _, elem := range list.Items {
		m := elem.ObjectMeta.Labels["meshName"]
		ns := elem.ObjectMeta.Namespace
		fmt.Fprintf(w, "%s\t%s\t\n", m, ns)
	}
	w.Flush()

	return nil
}

func (l *meshListCmd) selectMeshes() (*v1.DeploymentList, error) {
	deploymentsClient := l.clientSet.AppsV1().Deployments("") // Get deployments from all namespaces
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	return deploymentsClient.List(context.TODO(), listOptions)
}

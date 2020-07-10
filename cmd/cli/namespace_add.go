package main

import (
	"context"
	"fmt"
	"io"

	"github.com/open-service-mesh/osm/pkg/constants"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const namespaceAddDescription = `
This command will join a namespace to the mesh. All services
in this namespace will be part of the mesh.

`

type namespaceAddCmd struct {
	out       io.Writer
	namespace string
	meshName  string
	clientSet kubernetes.Interface
}

func newNamespaceAdd(out io.Writer) *cobra.Command {
	namespaceAdd := &namespaceAddCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "add <NAMESPACE>",
		Short: "add namespace to mesh",
		Long:  namespaceAddDescription,
		Args:  require.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			namespaceAdd.namespace = args[0]
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			namespaceAdd.clientSet = clientset
			return namespaceAdd.run()
		},
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVar(&namespaceAdd.meshName, "mesh-name", "osm", "Name of the service mesh")

	return cmd
}

func (a *namespaceAddCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	patch := `{"metadata":{"labels":{"` + constants.OSMKubeResourceMonitorAnnotation + `":"` + a.meshName + `"}}}`
	_, err := a.clientSet.CoreV1().Namespaces().Patch(ctx, a.namespace, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
	if err != nil {
		return fmt.Errorf("Could not label namespace [%s]: %v", a.namespace, err)
	}

	fmt.Fprintf(a.out, "Namespace [%s] succesfully added to mesh [%s]\n", a.namespace, a.meshName)

	return nil
}

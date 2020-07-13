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

const namespaceRemoveDescription = `
This command will remove a namespace from the mesh. All 
services in this namespace will be removed from the mesh.

`

type namespaceRemoveCmd struct {
	out       io.Writer
	namespace string
	meshName  string
	clientSet kubernetes.Interface
}

func newNamespaceRemove(out io.Writer) *cobra.Command {
	namespaceRemove := &namespaceRemoveCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "remove <NAMESPACE>",
		Short: "remove namespace from mesh",
		Long:  namespaceRemoveDescription,
		Args:  require.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			namespaceRemove.namespace = args[0]
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return fmt.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return fmt.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			namespaceRemove.clientSet = clientset
			return namespaceRemove.run()
		},
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVar(&namespaceRemove.meshName, "mesh-name", "osm", "Name of the service mesh")

	return cmd
}

func (r *namespaceRemoveCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	namespace, err := r.clientSet.CoreV1().Namespaces().Get(ctx, r.namespace, metav1.GetOptions{})

	if err != nil {
		return fmt.Errorf("Could not get namespace [%s]: %v", r.namespace, err)
	}

	val, exists := namespace.ObjectMeta.Labels[constants.OSMKubeResourceMonitorAnnotation]
	if exists {
		if val == r.meshName {
			patch := `{"metadata":{"labels":{"$patch":"delete", "` + constants.OSMKubeResourceMonitorAnnotation + `":"` + r.meshName + `"}}}`

			_, err = r.clientSet.CoreV1().Namespaces().Patch(ctx, r.namespace, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")

			if err != nil {
				return fmt.Errorf("Could not remove label from namespace %s: %v", r.namespace, err)
			}

			fmt.Fprintf(r.out, "Namespace [%s] succesfully removed from mesh [%s]\n", r.namespace, r.meshName)
		} else {
			return fmt.Errorf("Namespace belongs to mesh [%s], not mesh [%s]. Please specify the correct mesh", val, r.meshName)
		}
	} else {
		fmt.Fprintf(r.out, "Namespace [%s] already does not belong to any mesh\n", r.namespace)
		return nil
	}

	return nil
}

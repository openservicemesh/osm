package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
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
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			namespaceRemove.namespace = args[0]
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
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
		return errors.Errorf("Could not get namespace [%s]: %v", r.namespace, err)
	}

	val, exists := namespace.ObjectMeta.Labels[constants.OSMKubeResourceMonitorAnnotation]
	if exists {
		if val == r.meshName {
			// Setting null for a key in a map removes only that specific key, which is the desired behavior.
			// Even if the key does not exist, there will be no side effects with setting the key to null, which
			// will result in the same behavior as if the key were present - the key being removed.
			patch := fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": null
		},
		"annotations": {
			"%s": null
		}
	}
}`, constants.OSMKubeResourceMonitorAnnotation, constants.SidecarInjectionAnnotation)

			_, err = r.clientSet.CoreV1().Namespaces().Patch(ctx, r.namespace, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")

			if err != nil {
				return errors.Errorf("Could not remove namespace [%s] from mesh [%s]: %v", r.namespace, r.meshName, err)
			}

			fmt.Fprintf(r.out, "Namespace [%s] successfully removed from mesh [%s]\n", r.namespace, r.meshName)
		} else {
			return errors.Errorf("Namespace belongs to mesh [%s], not mesh [%s]. Please specify the correct mesh", val, r.meshName)
		}
	} else {
		fmt.Fprintf(r.out, "Namespace [%s] already does not belong to any mesh\n", r.namespace)
		return nil
	}

	return nil
}

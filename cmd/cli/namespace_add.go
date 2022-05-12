package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/constants"
)

const trueValue = "true"

const namespaceAddDescription = `
This command adds a namespace or set of namespaces to the mesh so that the osm
control plane with the given mesh name can observe resources within that namespace
or set of namespaces. It also enables automatic sidecar injection for all pods
created within the given namespace. Automatic sidecar injection can be disabled
via the --disable-sidecar-injection flag.
`
const namespaceAddExample = `
# Add namespace 'test' to the mesh with automatic sidecar injection enabled.
osm namespace add test

# Add namespace 'test' to the mesh while disabling automatic sidecar injection.
# If sidecar injection was previously enabled, it will be disabled by this command.
osm namespace add test --disable-sidecar-injection

# Add multiple namespaces (test, foo, bar, baz) to the mesh at the same time.
osm namespace add test foo bar baz

# Specify which mesh (osm control plane) to add the namespace if multiple control planes
are present or mesh name was overridden at install time
osm namespace add test --mesh-name=<my-mesh-name>
`

type namespaceAddCmd struct {
	out                     io.Writer
	namespaces              []string
	meshName                string
	disableSidecarInjection bool
	clientSet               kubernetes.Interface
}

func newNamespaceAdd(out io.Writer) *cobra.Command {
	namespaceAdd := &namespaceAddCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "add NAMESPACE ...",
		Short: "add namespace to mesh",
		Long:  namespaceAddDescription,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespaceAdd.namespaces = args
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}
			namespaceAdd.clientSet = clientset
			return namespaceAdd.run()
		},
		Example: namespaceAddExample,
	}

	//add mesh name flag
	f := cmd.Flags()
	f.StringVar(&namespaceAdd.meshName, "mesh-name", "osm", "Name of the service mesh")

	//add sidecar injection flag
	f.BoolVar(&namespaceAdd.disableSidecarInjection, "disable-sidecar-injection", false, "Disable automatic sidecar injection")

	return cmd
}

func (a *namespaceAddCmd) run() error {
	for _, ns := range a.namespaces {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		exists, err := meshExists(a.clientSet, a.meshName)
		if err != nil {
			return err
		}
		if !exists {
			return errors.Errorf("mesh [%s] does not exist, please specify another mesh using --mesh-name or create a new mesh", a.meshName)
		}

		deploymentsClient := a.clientSet.AppsV1().Deployments(ns)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{constants.AppLabel: constants.OSMControllerName}}

		listOptions := metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		}
		list, _ := deploymentsClient.List(context.TODO(), listOptions)

		// if osm-controller is installed in this namespace then don't add that to mesh
		if len(list.Items) != 0 {
			_, _ = fmt.Fprintf(a.out, "Namespace [%s] already has [%s] installed and cannot be added to mesh [%s]\n", ns, constants.OSMControllerName, a.meshName)
			continue
		}

		// if the namespace is already a part of the mesh then don't add it again
		namespace, err := a.clientSet.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			return errors.Errorf("Could not add namespace [%s] to mesh [%s]: %v", ns, a.meshName, err)
		}
		meshName := namespace.Labels[constants.OSMKubeResourceMonitorAnnotation]
		if a.meshName == meshName {
			_, _ = fmt.Fprintf(a.out, "Namespace [%s] has already been added to mesh [%s]\n", ns, a.meshName)
			continue
		}

		// if ignore label exits don`t add namespace
		if val, ok := namespace.ObjectMeta.Labels[constants.IgnoreLabel]; ok && val == trueValue {
			return errors.Errorf("Cannot add ignored namespace")
		}

		var patch string
		if a.disableSidecarInjection {
			// Patch the namespace with monitoring label.
			// Disable sidecar injection.
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		},
		"annotations": {
			"%s": "disabled"
		}
	}
}`, constants.OSMKubeResourceMonitorAnnotation, a.meshName, constants.SidecarInjectionAnnotation)
		} else {
			// Patch the namespace with the monitoring label.
			// Enable sidecar injection.
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		},
		"annotations": {
			"%s": "enabled"
		}
	}
}`, constants.OSMKubeResourceMonitorAnnotation, a.meshName, constants.SidecarInjectionAnnotation)
		}

		_, err = a.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return errors.Errorf("Could not add namespace [%s] to mesh [%s]: %v", ns, a.meshName, err)
		}

		_, _ = fmt.Fprintf(a.out, "Namespace [%s] successfully added to mesh [%s]\n", ns, a.meshName)
	}

	return nil
}

// meshExists determines if a mesh with meshName exists within the cluster
func meshExists(clientSet kubernetes.Interface, meshName string) (bool, error) {
	// search for the mesh across all namespaces
	deploymentsClient := clientSet.AppsV1().Deployments("")
	// search and match using the mesh name provided
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"meshName": meshName}}
	listOptions := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	osmControllerDeployments, err := deploymentsClient.List(context.TODO(), listOptions)
	if err != nil {
		return false, errors.Errorf("Cannot obtain information about the mesh [%s]: [%v]", meshName, err)
	}
	// the mesh is present if there are osm controllers for the mesh
	return len(osmControllerDeployments.Items) != 0, nil
}

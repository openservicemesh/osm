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

const namespaceAddDescription = `
This command will add a namespace or a set of namespaces 
to the mesh, automatic sidecar injection is disabled by 
default.Optionally, the namespaces can be configured for 
automatic sidecar injection which enables pods in the 
added namespaces to be injected with a sidecar upon creation.
`
const namespaceAddExample = `
# Add bookstore to the mesh,disabled automatic sidecar injection.If it has been enabled, it will not be disabled.
osm namespace add bookstore

# Enable automatic sidecar injection
osm namespace add bookstore --enable-sidecar-injection 
osm namespace add bookstore --enable-sidecar-injection=true

# Disable automatic  sidecar injection
osm namespace add bookstore --enable-sidecar-injection=false`

type namespaceAddCmd struct {
	out                     io.Writer
	namespaces              []string
	meshName                string
	sidecarInjectionFlagSet bool
	enableSidecarInjection  bool
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
			namespaceAdd.sidecarInjectionFlagSet = cmd.Flags().Changed("enable-sidecar-injection")
			config, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
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
	f.BoolVar(&namespaceAdd.enableSidecarInjection, "enable-sidecar-injection", true, "Enable/Disable automatic sidecar injection,explicitly specify this flag to have an effect.")

	return cmd
}

func (a *namespaceAddCmd) run() error {
	for _, ns := range a.namespaces {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		deploymentsClient := a.clientSet.AppsV1().Deployments(ns)
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{"app": constants.OSMControllerName}}

		listOptions := metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		}
		list, _ := deploymentsClient.List(context.TODO(), listOptions)

		// if osm-controller is installed in this namespace then don't add that to mesh
		if len(list.Items) != 0 {
			_, _ = fmt.Fprintf(a.out, "Namespace [%s] already has [%s] installed and cannot be added to mesh [%s]\n", ns, constants.OSMControllerName, a.meshName)
			continue
		}

		var patch string
		if !a.sidecarInjectionFlagSet {
			// Patch the namespace with monitoring label
			// Do not enable sidecar injection. If sidecar injection has been set, please do not disable it.
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		}
	}
}`, constants.OSMKubeResourceMonitorAnnotation, a.meshName)
		} else if a.enableSidecarInjection {
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
		} else {
			// Do not enable sidecar injection.
			// Remove annotations
			patch = fmt.Sprintf(`
{
	"metadata": {
		"labels": {
			"%s": "%s"
		},
		"annotations": {
			"%s": null
		}
	}
}`, constants.OSMKubeResourceMonitorAnnotation, a.meshName, constants.SidecarInjectionAnnotation)
		}
		_, err := a.clientSet.CoreV1().Namespaces().Patch(ctx, ns, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "")
		if err != nil {
			return errors.Errorf("Could not add namespace [%s] to mesh [%s]: %v", ns, a.meshName, err)
		}

		_, _ = fmt.Fprintf(a.out, "Namespace [%s] successfully added to mesh [%s]\n", ns, a.meshName)
	}

	return nil
}

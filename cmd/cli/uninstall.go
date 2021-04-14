package main

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const meshUninstallDescription = `
This command will uninstall an instance of the osm control plane
given the mesh name and namespace. It will not delete the namespace
the mesh was installed in.
Only use this in non-production and test environments.
`

type uninstallCmd struct {
	out             io.Writer
	in              io.Reader
	meshName        string
	force           bool
	deleteNamespace bool
	client          *action.Uninstall
	clientSet       kubernetes.Interface
}

func newUninstallCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &uninstallCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"delete", "del"},
		Short:   "uninstall osm control plane instance",
		Long:    meshUninstallDescription,
		Args:    cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			uninstall.client = action.NewUninstall(config)

			// get kubeconfig and initialize k8s client
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			uninstall.clientSet, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}

			return uninstall.run()
		},
	}

	f := cmd.Flags()
	//add mesh name flag
	f.StringVar(&uninstall.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	//add force uninstall flag
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the osm control plane instance without prompting for confirmation.  If the control plane with specified mesh name does not exist, do not display a diagnostic message or modify the exit status to reflect an error.")
	//add uninstall namespace flag
	f.BoolVar(&uninstall.deleteNamespace, "delete-namespace", false, "Attempt to delete the namespace after control plane components are deleted")

	return cmd
}

func (d *uninstallCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns := settings.Namespace()

	if !d.force {
		confirm, err := confirm(d.in, d.out, fmt.Sprintf("Uninstall OSM [mesh name: %s] ?", d.meshName), 3)
		if !confirm || err != nil {
			return err
		}
	}

	_, err := d.client.Run(d.meshName)
	if err != nil && errors.Cause(err) == helmStorage.ErrReleaseNotFound {
		if d.force {
			return nil
		}
		return errors.Errorf("No OSM control plane with mesh name [%s] found in namespace [%s]", d.meshName, ns)
	}

	if err == nil {
		fmt.Fprintf(d.out, "OSM [mesh name: %s] uninstalled\n", d.meshName)
	}

	if d.deleteNamespace {
		if err = d.clientSet.CoreV1().Namespaces().Delete(ctx, ns, v1.DeleteOptions{}); err != nil {
			return errors.Errorf("Error occurred while deleting OSM namespace [%s] - %v", ns, err)
		}
		fmt.Fprintf(d.out, "OSM namespace [%s] deleted successfully\n", ns)
	}

	return err
}

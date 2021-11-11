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
	"k8s.io/client-go/rest"

	"github.com/openservicemesh/osm/pkg/constants"
)

const uninstallMeshDescription = `
This command will uninstall an instance of the osm control plane
given the mesh name and namespace. It will not delete the namespace
the mesh was installed in unless specified via the --delete-namespace
flag.

Uninstalling OSM (through 'osm uninstall mesh' or through Helm) will only:
(1) remove osm control plane components (including control plane pods and secrets),
(2) remove/un-patch the conversion webhook fields from all the CRDs
(which OSM adds to support multiple CR versions) and will not delete
CRDs for primarily two reasons:
	1. CRDs are cluster-wide resources and may be used by other service meshes
	or resources running in the same cluster,
	2. deletion of a CRD will cause all custom resources corresponding to
	that CRD to also be deleted.

CRDs, mutating and validating webhooks, secrets, and the osm control
plane namespace are left over when this command is run.

Use 'osm uninstall cluster-wide-resources' to fully uninstall
leftover osm resources from the cluster.

Be careful when using this command as it is destructive and will
disrupt traffic to applications left running with sidecar proxies.
`

type uninstallMeshCmd struct {
	out             io.Writer
	in              io.Reader
	config          *rest.Config
	meshName        string
	force           bool
	deleteNamespace bool
	client          *action.Uninstall
	clientSet       kubernetes.Interface
	localPort       uint16
}

func newUninstallMeshCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	uninstall := &uninstallMeshCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "uninstall osm control plane instance",
		Long:  uninstallMeshDescription,
		Args:  cobra.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			uninstall.client = action.NewUninstall(config)

			// get kubeconfig and initialize k8s client
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig: %s", err)
			}
			uninstall.config = kubeconfig

			uninstall.clientSet, err = kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster, check kubeconfig: %s", err)
			}

			return uninstall.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&uninstall.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	f.BoolVarP(&uninstall.force, "force", "f", false, "Attempt to uninstall the osm control plane instance without prompting for confirmation.  If the control plane with specified mesh name does not exist, do not display a diagnostic message or modify the exit status to reflect an error.")
	f.BoolVar(&uninstall.deleteNamespace, "delete-namespace", false, "Attempt to delete the namespace after control plane components are deleted")
	f.Uint16VarP(&uninstall.localPort, "local-port", "p", constants.OSMHTTPServerPort, "Local port to use for port forwarding")

	return cmd
}

func (d *uninstallMeshCmd) run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns := settings.Namespace()

	if !d.force {
		// print a list of meshes within the cluster for a better user experience
		fmt.Fprintf(d.out, "\nList of meshes present in the cluster:\n")

		listCmd := &meshListCmd{
			out:       d.out,
			config:    d.config,
			clientSet: d.clientSet,
			localPort: d.localPort,
		}

		_ = listCmd.run()

		confirm, err := confirm(d.in, d.out, fmt.Sprintf("\nUninstall OSM [mesh name: %s] in namespace [%s] ?", d.meshName, ns), 3)
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
		fmt.Fprintf(d.out, "OSM [mesh name: %s] in namespace [%s] uninstalled\n", d.meshName, ns)
	}

	if d.deleteNamespace {
		if err = d.clientSet.CoreV1().Namespaces().Delete(ctx, ns, v1.DeleteOptions{}); err != nil {
			return errors.Errorf("Error occurred while deleting OSM namespace [%s] - %v", ns, err)
		}
		fmt.Fprintf(d.out, "OSM namespace [%s] deleted successfully\n", ns)
	}

	return err
}

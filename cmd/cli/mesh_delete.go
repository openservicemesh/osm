package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
	"io"
)

const meshDeleteDescription = `
This command will delete an instance of the osm control plane
given the mesh name and namespace. It will not delete the namespace
the mesh was installed in.

Only use this in non-production and test environments.
`

type meshDeleteCmd struct {
	out      io.Writer
	in       io.Reader
	meshName string
	force    bool
	client   *action.Uninstall
}

func newMeshDelete(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	del := &meshDeleteCmd{
		out: out,
		in:  in,
	}

	cmd := &cobra.Command{
		Use:   "delete MESH_NAME",
		Short: "delete osm control plane instance",
		Long:  meshDeleteDescription,
		Args:  require.ExactArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			del.client = action.NewUninstall(config)
			return del.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&del.meshName, "mesh-name", defaultMeshName, "Name of the service mesh")
	f.BoolVarP(&del.force, "force", "f", false, "Attempt to delete the osm control plane instance without prompting for confirmation.  If the control plane with specified mesh name does not exist, do not display a diagnostic message or modify the exit status to reflect an error.")

	return cmd
}

func (d *meshDeleteCmd) run() error {
	if d.force {
		_, _ = d.client.Run(d.meshName)
		return nil
	}
	confirm, err := confirm(d.in, d.out, fmt.Sprintf("Delete OSM [mesh name: %s] ?", d.meshName), 3)
	if !confirm || err != nil {
		return err
	}

	_, err = d.client.Run(d.meshName)
	if err != nil && errors.Cause(err) == helmStorage.ErrReleaseNotFound {
		return errors.Errorf("No OSM control plane with mesh name [%s] found in namespace [%s]", d.meshName, settings.Namespace())
	}

	fmt.Fprintf(d.out, "OSM [mesh name: %s] deleted\n", d.meshName)

	return err
}

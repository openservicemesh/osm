package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
)

const meshDeleteDescription = `
This command will delete an instance of the osm control plane
given the mesh name and namespace. It will not delete the namespace
the mesh was installed in.

Only use this in non-production and test environments.
`

type meshDeleteCmd struct {
	out    io.Writer
	name   string
	client *action.Uninstall
}

func newMeshDelete(config *action.Configuration, out io.Writer) *cobra.Command {
	del := &meshDeleteCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete MESH_NAME",
		Short: "delete osm control plane instance",
		Long:  meshDeleteDescription,
		Args:  require.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			del.name = args[0]
			del.client = action.NewUninstall(config)
			return del.run()
		},
	}

	return cmd
}

func (d *meshDeleteCmd) run() error {

	_, err := d.client.Run(d.name)
	if err != nil && errors.Cause(err) == helmStorage.ErrReleaseNotFound {
		return errors.Errorf("No OSM control plane with mesh name [%s] found in namespace [%s]", d.name, settings.Namespace())
	}

	fmt.Fprintf(d.out, "OSM [mesh name: %s] deleted\n", d.name)

	return err
}

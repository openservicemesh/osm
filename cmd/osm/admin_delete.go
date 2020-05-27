package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
)

const adminDeleteOsmDescription = `
This command will delete an instance of the osm control plane
given the OSM ID and namespace of the control plane. It will
not delete the namespace that the control plane components live
in.

Only use this in non-production and test environments.

Usage:
  $ osm admin delete-osm [OSM ID] --namespace osm-system
`

type adminDeleteOsm struct {
	out       io.Writer
	osmID     string
	namespace string
	client    *action.Uninstall
}

func newAdminDelete(config *action.Configuration, out io.Writer) *cobra.Command {
	del := &adminDeleteOsm{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete-osm [OSM ID]",
		Short: "delete osm instance",
		Long:  adminDeleteOsmDescription,
		Args:  require.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			del.osmID = args[0]
			del.client = action.NewUninstall(config)
			return del.run()
		},
	}

	return cmd
}

func (d *adminDeleteOsm) run() error {

	_, err := d.client.Run(d.osmID)
	return err
}

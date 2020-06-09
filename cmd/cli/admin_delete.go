package main

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	helmStorage "helm.sh/helm/v3/pkg/storage/driver"
)

const adminDeleteOsmDescription = `
This command will delete an instance of the osm control plane
given the namespace of the osm control plane. It will
not delete the namespace itelf.

Only use this in non-production and test environments.

Usage:
  $ osm admin delete-osm --namespace osm-system
`

type adminDeleteOsm struct {
	out    io.Writer
	client *action.Uninstall
}

func newAdminDelete(config *action.Configuration, out io.Writer) *cobra.Command {
	del := &adminDeleteOsm{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "delete-osm",
		Short: "delete osm control plane",
		Long:  adminDeleteOsmDescription,
		RunE: func(_ *cobra.Command, args []string) error {
			del.client = action.NewUninstall(config)
			return del.run()
		},
	}

	return cmd
}

func (d *adminDeleteOsm) run() error {

	_, err := d.client.Run(settings.Namespace())
	if err != nil && errors.Cause(err) == helmStorage.ErrReleaseNotFound {
		return errors.Errorf("No OSM control plane found in namespace %s", settings.Namespace())
	}

	return err
}

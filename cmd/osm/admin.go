package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
)

const adminDescription = `
This command consists of multiple subcommands related to administrating instances
of osm installations.

`

func newAdminCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "admin osm installs",
		Long:  adminDescription,
		Args:  require.NoArgs,
	}
	cmd.AddCommand(newAdminDelete(config, out))

	return cmd
}

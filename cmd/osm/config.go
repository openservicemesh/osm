package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
)

const configDescription = `
This command consists of multiple subcommands which can be used to
configure OSM.
`

func newConfigCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "configure osm environment",
		Long:  configDescription,
		Args:  require.NoArgs,
	}
	cmd.AddCommand(newConfigACRSecretCmd(config, out))

	return cmd
}

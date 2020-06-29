package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
)

const meshDescription = `
This command consists of multiple subcommands related to managing instances of
osm installations. Each osm installation results in a mesh. Each installation
receives a unqiue mesh name.

`

func newMeshCmd(config *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "manage osm installations",
		Long:  meshDescription,
		Args:  require.NoArgs,
	}
	cmd.AddCommand(newMeshDelete(config, out))

	return cmd
}

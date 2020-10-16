package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const meshDescription = `
This command consists of multiple subcommands related to managing instances of
osm installations. Each osm installation results in a mesh. Each installation
receives a unique mesh name.

`

func newMeshCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "manage osm installations",
		Long:  meshDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newMeshUninstall(config, in, out))
	cmd.AddCommand(newMeshList(out))

	return cmd
}

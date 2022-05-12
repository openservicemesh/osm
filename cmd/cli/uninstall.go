package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const uninstallDescription = `
This command consists of multiple subcommands related to uninstalling the osm
control plane.
`

func newUninstallCmd(config *action.Configuration, in io.Reader, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "uninstall osm",
		Aliases: []string{"delete", "del"},
		Long:    uninstallDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newUninstallMeshCmd(config, in, out))

	return cmd
}

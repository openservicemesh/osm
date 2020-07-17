package main

import (
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
)

const namespaceDescription = `
This command consists of multiple subcommands related to managing namespaces
associated with osm installations.

`

func newNamespaceCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "manage osm namespaces",
		Long:  namespaceDescription,
		Args:  require.NoArgs,
	}
	cmd.AddCommand(newNamespaceAdd(out))
	cmd.AddCommand(newNamespaceRemove(out))
	cmd.AddCommand(newNamespaceList(out))

	return cmd
}

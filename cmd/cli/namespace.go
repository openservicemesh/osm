package main

import (
	"io"

	"github.com/spf13/cobra"
)

const namespaceDescription = `
This command consists of multiple subcommands related to managing namespaces
associated with osm installations.
`

func newNamespaceCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "namespace",
		Short:   "manage osm namespaces",
		Aliases: []string{"ns"},
		Long:    namespaceDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newNamespaceAdd(out))
	cmd.AddCommand(newNamespaceRemove(out))
	cmd.AddCommand(newNamespaceIgnore(out))
	cmd.AddCommand(newNamespaceList(out))

	return cmd
}

package main

import (
	"io"

	"github.com/spf13/cobra"
)

const certificateDescription = `
This command consists of multiple subcommands related to managing certificates
associated with osm installations.
`

func newCertificateCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "certificate",
		Short:   "commands for managing MeshRootCertificates",
		Aliases: []string{"mrc"},
		Long:    certificateDescription,
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(newCertificateRotateCmd(out))
	return cmd
}

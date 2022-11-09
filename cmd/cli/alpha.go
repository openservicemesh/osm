package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const alphaDescription = `
This command consists of multiple subcommands related that are in alpha.
`

func newAlphaCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alpha",
		Short: "commands that are in alpha",
		Long:  alphaDescription,
		Args:  cobra.NoArgs,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(out, "*** This command is in Preview.  Only run in dev/test environments ***\n")
		},
	}
	cmd.AddCommand(newCertificateCmd(out))
	return cmd
}

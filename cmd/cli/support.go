package main

import (
	"io"

	"github.com/spf13/cobra"
)

const supportCmdDescription = `
This command consists of subcommands related supportability and
associated tooling, such as examining error codes.
`

func newSupportCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "support",
		Short: "supportability tooling",
		Long:  supportCmdDescription,
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newSupportErrInfoCmd(out))

	return cmd
}

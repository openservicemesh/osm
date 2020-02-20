package main

import (
	goflag "flag"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var globalUsage = `smc enables you to install and manage the 
service mesh controller in your Kubernetes cluster

To install and configure service mesh controller, run:

   $ smc install
`

func newRootCmd(args []string, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smc",
		Short: "Install and manage service mesh controller",
		Long:  globalUsage,
	}

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()

	// Add subcommands here
	cmd.AddCommand(
		newInstallCmd(out),
	)

	flags.Parse(args)

	return cmd
}

func main() {
	cmd := newRootCmd(os.Args[1:], os.Stdout)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

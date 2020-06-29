package main

import (
	goflag "flag"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"

	"github.com/open-service-mesh/osm/pkg/cli"
)

var globalUsage = `osm enables you to install and manage the
Open Service Mesh in your Kubernetes cluster

To install and configure open service mesh, run:

   $ osm install
`

var settings = cli.New()

func newRootCmd(config *action.Configuration, out io.Writer, args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "osm",
		Short:        "Install and manage open service mesh",
		Long:         globalUsage,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()
	settings.AddFlags(flags)

	// Add subcommands here
	cmd.AddCommand(
		newMeshCmd(config, out),
		newEnvCmd(out),
		newInstallCmd(config, out),
		newCheckCmd(out),
	)

	flags.Parse(args)

	return cmd
}

func main() {
	actionConfig := new(action.Configuration)
	cmd := newRootCmd(actionConfig, os.Stdout, os.Args[1:])
	actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug)

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug); err != nil {
			os.Exit(1)
		}
	})

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func debug(format string, v ...interface{}) {
	format = fmt.Sprintf("[debug] %s\n", format)
}

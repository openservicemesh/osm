package main

import (
	goflag "flag"
	"io"
	"os"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"

	"github.com/openservicemesh/osm/pkg/cli"
)

var globalUsage = `The osm cli enables you to install and manage the
Open Service Mesh (OSM) in your Kubernetes cluster

To install and configure OSM, run:

   $ osm install
`

var settings = cli.New()

func newRootCmd(config *action.Configuration, in io.Reader, out io.Writer, args []string) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "osm",
		Short:        "Install and manage Open Service Mesh",
		Long:         globalUsage,
		SilenceUsage: true,
	}

	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)
	flags := cmd.PersistentFlags()
	settings.AddFlags(flags)

	// Add subcommands here
	cmd.AddCommand(
		newMeshCmd(config, in, out),
		newEnvCmd(out),
		newInstallCmd(config, out),
		newDashboardCmd(config, out),
		newNamespaceCmd(out),
		newMetricsCmd(out),
		newVersionCmd(out),
		newProxyCmd(config, out),
		newTrafficPolicyCmd(out),
	)

	_ = flags.Parse(args)

	return cmd
}

func main() {
	actionConfig := new(action.Configuration)
	cmd := newRootCmd(actionConfig, os.Stdin, os.Stdout, os.Args[1:])
	_ = actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug)

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
}

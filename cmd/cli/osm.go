// Package main implements OSM CLI commands and utility routines required by the CLI.
package main

import (
	goflag "flag"
	"fmt"
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

func newRootCmd(config *action.Configuration, stdin io.Reader, stdout io.Writer, stderr io.Writer, args []string) *cobra.Command {
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
		newMeshCmd(config, stdin, stdout),
		newEnvCmd(stdout, stderr),
		newNamespaceCmd(stdout),
		newMetricsCmd(stdout),
		newVersionCmd(stdout),
		newProxyCmd(config, stdout),
		newPolicyCmd(stdout, stderr),
		newSupportCmd(config, stdout, stderr),
		newUninstallCmd(config, stdin, stdout),
		newVerifyCmd(stdout, stderr),
	)

	// Add subcommands related to unmanaged environments
	if !settings.IsManaged() {
		cmd.AddCommand(
			newInstallCmd(config, stdout),
			newDashboardCmd(config, stdout),
		)
	}

	_ = flags.Parse(args)

	return cmd
}

func initCommands() *cobra.Command {
	actionConfig := new(action.Configuration)
	cmd := newRootCmd(actionConfig, os.Stdin, os.Stdout, os.Stderr, os.Args[1:])
	_ = actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug)

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), "secret", debug); err != nil {
			os.Exit(1)
		}
	})

	return cmd
}

func main() {
	cmd := initCommands()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func debug(format string, v ...interface{}) {
	if settings.Verbose() {
		format = fmt.Sprintf("[debug] %s\n", format)
		fmt.Printf(format, v...)
	}
}

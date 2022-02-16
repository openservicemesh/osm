package main

import (
	health_cmd "github.com/openservicemesh/osm-health/cmd"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
)

const healthDescription = `
This command consists of multiple subcommands related to managing namespaces
associated with osm installations.
`

var healthCmdName = "health"

func newHealthCmd(config *action.Configuration) *cobra.Command {
	return health_cmd.NewRootCmd(healthCmdName, config, []string{})
}
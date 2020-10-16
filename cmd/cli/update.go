package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/openservicemesh/osm/pkg/cli"
)

const updateDesc = `
This command updates an osm control plane on the Kubernetes cluster.

The default Kubernetes namespace that gets updated is called osm-system. 
To update a control plane components in a different namespace, use the 
--namespace flag.

Example:
  $ osm update --namespace hello-world --enable-egress false
`

type updateCmd struct {
	out io.Writer

	chartPath                     string
	enablePermissiveTrafficPolicy bool
	enableEgress                  bool
	meshCIDRRanges                []string
}

func newUpdateCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	upd := &updateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "update osm control plane configuration",
		Long:  updateDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return upd.run(config)
		},
	}

	f := cmd.Flags()

	f.StringVar(&upd.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.BoolVar(&upd.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", false, "Enable permissive traffic policy mode")
	f.BoolVar(&upd.enableEgress, "enable-egress", false, "Enable egress in the mesh")
	f.StringSliceVar(&upd.meshCIDRRanges, "mesh-cidr", []string{}, "mesh CIDR range, accepts multiple CIDRs, required if enable-egress option is true")

	return cmd
}

func (u *updateCmd) run(config *helm.Configuration) error {
	var chartRequested *chart.Chart
	var err error
	if u.chartPath != "" {
		chartRequested, err = loader.Load(u.chartPath)
	} else {
		chartRequested, err = cli.LoadChart(chartTGZSource)
	}
	if err != nil {
		return err
	}

	// Fetch the current release from Helm
	// This should exist for the upgrade to succeed.
	// If a release is not found an error is returned
	existingChart, err := helm.NewGet(config).Run(chartRequested.Name())
	if err != nil {
		return err
	}

	// Add the overlay values to be updated to the current release's values map
	values, err := u.resolveValues(existingChart.Chart.Values)
	if err != nil {
		return err
	}

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Namespace = settings.Namespace()
	if _, err = upgradeClient.Run(chartRequested.Name(), chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "OSM successfully updated in namespace [%s]\n", settings.Namespace())
	return nil
}

func (u *updateCmd) resolveValues(currentValues map[string]interface{}) (map[string]interface{}, error) {
	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.enablePermissiveTrafficPolicy=%t", u.enablePermissiveTrafficPolicy),
		fmt.Sprintf("OpenServiceMesh.enableEgress=%t", u.enableEgress),
		fmt.Sprintf("OpenServiceMesh.meshCIDRRanges=%s", strings.Join(u.meshCIDRRanges, " ")),
	}

	for _, val := range valuesConfig {
		if err := strvals.ParseInto(val, currentValues); err != nil {
			return currentValues, err
		}
	}

	return currentValues, nil
}

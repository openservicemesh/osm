package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
)

const upgradeDesc = `
This command upgrades an OSM control plane by upgrading the
underlying Helm release.

The mesh to upgrade is identified by its mesh name and namespace. If either were
overridden from the default for the "osm install" command, the --mesh-name and
--osm-namespace flags need to be specified.

Values from the current Helm release will NOT be carried over to the new
release. Use --set to pass any overridden values from the old release to the new
release.

Note: edits to resources NOT made by Helm or the OSM CLI may not persist after
"osm mesh upgrade" is run.

Note: edits made to chart values that impact the preset-mesh-config will not
apply to the osm-mesh-config, when "osm mesh upgrade" is run. This means configuration
changes made to the osm-mesh-config resource will persist through an upgrade
and any configuration changes needed can be done by patching this resource prior or
post an upgrade.

If any CustomResourceDefinitions (CRDs) are different between the installed
chart and the upgraded chart, the CRDs will be updated to include the latest versions.
Any corresponding custom resources that wish to reference the newer CRD version can
be updated post upgrade.
`

const meshUpgradeExample = `
# Upgrade the mesh with the default name in the osm-system namespace, setting
# the image registry and tag to the defaults, and leaving all other values unchanged.
osm mesh upgrade --osm-namespace osm-system
`

type meshUpgradeCmd struct {
	out io.Writer

	meshName string
	chart    *chart.Chart

	setOptions []string
}

func newMeshUpgradeCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	upg := &meshUpgradeCmd{
		out: out,
	}
	var chartPath string

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "upgrade osm control plane",
		Long:    upgradeDesc,
		Example: meshUpgradeExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			if chartPath != "" {
				var err error
				upg.chart, err = loader.Load(chartPath)
				if err != nil {
					return err
				}
			}

			return upg.run(config)
		},
	}

	f := cmd.Flags()

	f.StringVar(&upg.meshName, "mesh-name", defaultMeshName, "Name of the mesh to upgrade")
	f.StringVar(&chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringArrayVar(&upg.setOptions, "set", nil, "Set arbitrary chart values (can specify multiple or separate values with commas: key1=val1,key2=val2)")

	return cmd
}

func (u *meshUpgradeCmd) run(config *helm.Configuration) error {
	if u.chart == nil {
		var err error
		u.chart, err = loader.LoadArchive(bytes.NewReader(chartTGZSource))
		if err != nil {
			return err
		}
	}

	// Add the overlay values to be updated to the current release's values map
	values, err := u.resolveValues()
	if err != nil {
		return err
	}

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Wait = true
	upgradeClient.Timeout = 5 * time.Minute
	upgradeClient.ResetValues = true
	if _, err = upgradeClient.Run(u.meshName, u.chart, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "OSM successfully upgraded mesh [%s] in namespace [%s]\n", u.meshName, settings.Namespace())
	return nil
}

func (u *meshUpgradeCmd) resolveValues() (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for _, val := range u.setOptions {
		if err := strvals.ParseInto(val, vals); err != nil {
			return nil, errors.Wrap(err, "invalid format for --set")
		}
	}
	return vals, nil
}

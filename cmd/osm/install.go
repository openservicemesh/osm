package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
)

const installDesc = `
This command installs the osm control plane on the Kubernetes cluster.
`
const (
	defaultOSMChartPath    = "charts/osm"
	defaultHelmReleaseName = "osm-cp"
)

type installCmd struct {
	out                     io.Writer
	containerRegistry       string
	containerRegistrySecret string
	chartPath               string
}

func newInstallCmd(config *helm.Configuration, out io.Writer) *cobra.Command {

	inst := &installCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install osm control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return inst.run(helm.NewInstall(config))
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.containerRegistry, "container-registry", "smctest.azurecr.io", "container registry that hosts control plane component images")
	f.StringVar(&inst.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&inst.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")

	return cmd
}

func (i *installCmd) run(installClient *helm.Install) error {
	installClient.ReleaseName = defaultHelmReleaseName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true

	chart := ""
	if i.chartPath != "" {
		chart = i.chartPath
	} else {
		chart = defaultOSMChartPath
	}
	chartRequested, err := loader.Load(chart)
	if err != nil {
		return err
	}

	values, err := i.resolveValues()
	if err != nil {
		return err
	}

	if _, err = installClient.Run(chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(i.out, "OSM installed successfully in %s namespace\n", settings.Namespace())
	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}
	valuesConfig := []string{
		fmt.Sprintf("image.registry=%s", i.containerRegistry),
		fmt.Sprintf("imagePullSecrets[0].name=%s", i.containerRegistrySecret),
		fmt.Sprintf("namespace=%s", settings.Namespace()),
	}

	for _, val := range valuesConfig {
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return finalValues, err
		}
	}
	return finalValues, nil
}

package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"
)

const installDesc = `
This command installs the osm control plane on the Kubernetes cluster.
`
const (
	defaultOSMChartPath = "charts/osm"
)

type installCmd struct {
	out                     io.Writer
	config                  *action.Configuration
	containerRegistry       string
	containerRegistrySecret string
}

func newInstallCmd(config *action.Configuration, out io.Writer) *cobra.Command {

	install := &installCmd{
		out:    out,
		config: config,
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "install osm control plane",
		Long:  installDesc,
		RunE: func(_ *cobra.Command, args []string) error {
			return install.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&install.containerRegistry, "container-registry", "smctest.azurecr.io", "container registry that hosts control plane component images")
	f.StringVar(&install.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")

	return cmd
}

func (i *installCmd) run() error {
	installClient := action.NewInstall(i.config)
	installClient.ReleaseName = "osm-cp"
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true

	chartRequested, err := loader.Load(defaultOSMChartPath)
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

	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}
	valuesConfig := []string{
		fmt.Sprintf("image.registry=%s", i.containerRegistry),
		fmt.Sprintf("imagePullSecrets[0].name=%s", i.containerRegistrySecret),
	}

	for _, val := range valuesConfig {
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return finalValues, err
		}
	}
	return finalValues, nil
}

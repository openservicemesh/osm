package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"helm.sh/helm/v3/cmd/helm/require"
	"helm.sh/helm/v3/pkg/action"
)

const configACRSecretDescription = `
This command can be used to generate a container registry secret for pulling OSM images.
It takes the name of an Azure container registry and creates a Kubernetes secret for it in a Kubernetes namespace.

The name of the Kubernetes secret is 'acr-creds' by default by can be configured via a flag. The secret will be installed into the 'osm-system' Kubernetes namespace by default. This can be re-configured via a flag as well. If the namespace does not exist, it will be created.

Example Usage:
  $ osm config acr-secret smctest.azurecr.io --secret-name acr-creds
`

type acrSecretCmd struct {
	out          io.Writer
	config       *action.Configuration
	registryName string
	secretName   string
}

func newConfigACRSecretCmd(config *action.Configuration, out io.Writer) *cobra.Command {

	acr := &acrSecretCmd{
		out:    out,
		config: config,
	}

	cmd := &cobra.Command{
		Use:   "acr-secret",
		Short: "configure ACR Kubernetes secret for pulling OSM images",
		Long:  configACRSecretDescription,
		Args:  require.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			acr.registryName = args[0]
			return acr.run()
		},
	}
	f := cmd.Flags()
	f.StringVar(&acr.secretName, "secret-name", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")

	return cmd
}

func (a *acrSecretCmd) run() error {
	cmd := exec.Command("kubectl", "create", "namespace", settings.Namespace())
	cmd.Run()

	fmt.Fprintf(a.out, "Creating Kubernetes secret [%s] for container registry [%s] credentials\n",
		a.secretName, a.registryName)

	registry := strings.Split(a.registryName, ".")[0]
	cmd = exec.Command("az", "acr", "credential", "show", "-n", registry, "--query", "passwords[0].value")
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.out.Write(output)
		return err
	}
	password := strings.Split(string(output), "\"")[1]

	cmd = exec.Command("kubectl", "create", "secret", "docker-registry", a.secretName,
		"-n", settings.Namespace(),
		"--docker-server", a.registryName,
		"--docker-username", registry,
		"--docker-email", "noone@example.com",
		"--docker-password", password,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {

		if !strings.Contains(string(output), "AlreadyExists") {
			a.out.Write(output)
			return err
		}
		fmt.Fprintf(a.out, "Kubernetes secret [%s] already exists\n", a.secretName)
	}
	return nil
}

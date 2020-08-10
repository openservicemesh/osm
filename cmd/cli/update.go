package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
)

const updateDesc = `
This command updates the osm control plane configuration running on a Kubernetes cluster.

The default Kubernetes namespace that gets updated is called osm-system. 
To update a control plane components in a different namespace, use the 
--namespace flag.

Example:
  $ osm update --namespace hello-world
`

type updateCmd struct {
	out       io.Writer
	clientSet kubernetes.Interface
	// action config is used to resolve the helm chart values
	cfg actionConfig
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
			kubeconfig, err := settings.RESTClientGetter().ToRESTConfig()
			if err != nil {
				return errors.Errorf("Error fetching kubeconfig")
			}

			clientset, err := kubernetes.NewForConfig(kubeconfig)
			if err != nil {
				return errors.Errorf("Could not access Kubernetes cluster. Check kubeconfig")
			}
			upd.clientSet = clientset
			return upd.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&upd.cfg.containerRegistry, "container-registry", "openservicemesh", "container registry that hosts control plane component images")
	f.StringVar(&upd.cfg.osmImageTag, "osm-image-tag", "v0.2.0", "osm image tag")
	f.StringVar(&upd.cfg.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&upd.cfg.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringVar(&upd.cfg.certManager, "certificate-manager", defaultCertManager, "certificate manager to use (tresor or vault)")
	f.StringVar(&upd.cfg.vaultHost, "vault-host", "", "Hashicorp Vault host/service - where Vault is installed")
	f.StringVar(&upd.cfg.vaultProtocol, "vault-protocol", defaultVaultProtocol, "protocol to use to connect to Vault")
	f.StringVar(&upd.cfg.vaultToken, "vault-token", "", "token that should be used to connect to Vault")
	f.StringVar(&upd.cfg.vaultRole, "vault-role", "openservicemesh", "Vault role to be used by Open Service Mesh")
	f.IntVar(&upd.cfg.serviceCertValidityMinutes, "service-cert-validity-minutes", defaultCertValidityMinutes, "Certificate TTL in minutes")
	f.StringVar(&upd.cfg.prometheusRetentionTime, "prometheus-retention-time", constants.PrometheusDefaultRetentionTime, "Duration for which data will be retained in prometheus")
	f.BoolVar(&upd.cfg.enableDebugServer, "enable-debug-server", false, "Enable the debug HTTP server")
	f.BoolVar(&upd.cfg.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", false, "Enable permissive traffic policy mode")
	f.BoolVar(&upd.cfg.enableEgress, "enable-egress", false, "Enable egress in the mesh")
	f.StringSliceVar(&upd.cfg.meshCIDRRanges, "mesh-cidr", []string{}, "mesh CIDR range, accepts multiple CIDRs, required if enable-egress option is true")
	f.BoolVar(&upd.cfg.enableBackpressureExperimental, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
	f.BoolVar(&upd.cfg.enableMetricsStack, "enable-metrics-stack", true, "Enable metrics (Prometheus and Grafana) deployment")
	f.StringVar(&upd.cfg.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")
	f.BoolVar(&upd.cfg.deployZipkin, "deploy-zipkin", true, "Deploy Zipkin in the namespace of the OSM controller")

	return cmd
}

func (u *updateCmd) run(config *helm.Configuration) error {
	var chartRequested *chart.Chart
	var err error
	if u.cfg.chartPath != "" {
		chartRequested, err = loader.Load(u.cfg.chartPath)
	} else {
		chartRequested, err = cli.LoadChart(chartTGZSource)
	}
	if err != nil {
		return err
	}

	err = u.cfg.validate()
	if err != nil {
		return err
	}

	values, err := u.cfg.resolveValues()
	if err != nil {
		return err
	}

	// TODO (nitishm): Should we check if the controlplane deployment exists ?

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Namespace = settings.Namespace()
	if _, err = upgradeClient.Run(chartRequested.Name(), chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "OSM successfully updated in namespace [%s] with mesh name [%s]\n", settings.Namespace(), u.cfg.meshName)
	return nil
}

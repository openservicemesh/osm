/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright 2020 The OSM contributors

Licensed under the MIT License
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

This file is inspired by the way Helm handles environment variables
for the Helm CLI https://github.com/helm/helm/blob/master/cmd/helm/env.go
*/
package main

import (
	"fmt"
	"io"

	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
)

const updateDesc = `
This command updates an osm control plane on the Kubernetes cluster.

The default Kubernetes namespace that gets updated is called osm-system. 
To update a control plane components in a different namespace, use the 
--namespace flag.

Example:
  $ osm update --namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "osm." The mesh name must conform to same
guidelines as a valid Kubernetes label value. Must be 63 characters or
less and must be empty or begin and end with an alphanumeric character
([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and
alphanumerics between.

Example:
  $ osm update --mesh-name "hello-osm"

The mesh name is used in various ways like for naming Kubernetes resources as
well as for adding a Kubernetes Namespace to the list of Namespaces a control
plane should watch for sidecar injection of Envoy proxies.
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

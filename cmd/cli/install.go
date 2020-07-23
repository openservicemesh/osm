package main

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/strvals"

	"github.com/open-service-mesh/osm/pkg/check"
	"github.com/open-service-mesh/osm/pkg/cli"
	"github.com/open-service-mesh/osm/pkg/constants"
	"k8s.io/apimachinery/pkg/util/validation"
)

const installDesc = `
This command installs an osm control plane on the Kubernetes cluster.

An osm control plane is comprised of namespaced Kubernetes resources
that get installed into the osm-system namespace as well as cluster
wide Kubernetes resources.

The default Kubernetes namespace that gets created on install is called
osm-system. To create an install control plane components in a different
namespace, use the --namespace flag.

Example:
  $ osm install --namespace hello-world

Multiple control plane installations can exist within a cluster. Each
control plane is given a cluster-wide unqiue identifier called mesh name.
A mesh name can be passed in via the --mesh-name flag. By default, the
mesh-name name will be set to "osm."

Example:
  $ osm install --mesh-name "hello-osm"

The mesh name is used in various ways like for naming Kubernetes resources as
well as for adding a Kubernetes Namespace to the list of Namespaces a control
plane should watch for sidecar injection of Envoy proxies.
`
const (
	defaultCertManager   = "tresor"
	defaultVaultProtocol = "http"
	defaultMeshName      = "osm"

	defaultCertValidityMinutes = int(1440) // 24 hours
)

// chartTGZSource is a base64-encoded, gzipped tarball of the default Helm chart.
// Its value is initialized at build time.
var chartTGZSource string

type installCmd struct {
	out                           io.Writer
	containerRegistry             string
	containerRegistrySecret       string
	chartPath                     string
	osmImageTag                   string
	certManager                   string
	vaultHost                     string
	vaultProtocol                 string
	vaultToken                    string
	vaultRole                     string
	serviceCertValidityMinutes    int
	prometheusRetentionTime       string
	enableDebugServer             bool
	enablePermissiveTrafficPolicy bool
	enableEgress                  bool
	meshName                      string
	meshCIDRRanges                []string

	// This is an experimental flag, which will eventually
	// become part of SMI Spec.
	enableBackpressureExperimental bool

	// Toggle to deploy/not deploy metrics (Promethus+Grafana) stack
	enableMetricsStack bool

	// checker runs checks before any installation is attempted. Its type is
	// abstract here to make testing easy.
	checker interface {
		Run([]check.Check, func(*check.Result)) bool
	}
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
			inst.checker = check.NewChecker(settings)
			return inst.run(config)
		},
	}

	f := cmd.Flags()
	f.StringVar(&inst.containerRegistry, "container-registry", "smctest.azurecr.io", "container registry that hosts control plane component images")
	f.StringVar(&inst.osmImageTag, "osm-image-tag", "latest", "osm image tag")
	f.StringVar(&inst.containerRegistrySecret, "container-registry-secret", "acr-creds", "name of Kubernetes secret for container registry credentials to be created if it doesn't already exist")
	f.StringVar(&inst.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringVar(&inst.certManager, "cert-manager", defaultCertManager, "certificate manager to use (tresor or vault)")
	f.StringVar(&inst.vaultHost, "vault-host", "", "Hashicorp Vault host/service - where Vault is installed")
	f.StringVar(&inst.vaultProtocol, "vault-protocol", defaultVaultProtocol, "protocol to use to connect to Vault")
	f.StringVar(&inst.vaultToken, "vault-token", "", "token that should be used to connect to Vault")
	f.StringVar(&inst.vaultRole, "vault-role", "open-service-mesh", "Vault role to be used by Open Service Mesh")
	f.IntVar(&inst.serviceCertValidityMinutes, "service-cert-validity-minutes", defaultCertValidityMinutes, "Certificate TTL in minutes")
	f.StringVar(&inst.prometheusRetentionTime, "prometheus-retention-time", constants.PrometheusDefaultRetentionTime, "Duration for which data will be retained in prometheus")
	f.BoolVar(&inst.enableDebugServer, "enable-debug-server", false, "Enable the debug HTTP server")
	f.BoolVar(&inst.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", false, "Enable permissive traffic policy mode")
	f.BoolVar(&inst.enableEgress, "enable-egress", true, "Enable egress in the mesh")
	f.StringSliceVar(&inst.meshCIDRRanges, "mesh-cidr", []string{}, "mesh CIDR range, accepts multiple CIDRs, required if enable-egress option is true")
	f.BoolVar(&inst.enableBackpressureExperimental, "enable-backpressure-experimental", false, "Enable experimental backpressure feature")
	f.BoolVar(&inst.enableMetricsStack, "enable-metrics-stack", true, "Enable metrics (Prometheus and Grafana) deployment")
	f.StringVar(&inst.meshName, "mesh-name", defaultMeshName, "name for the new control plane instance")

	return cmd
}

func (i *installCmd) run(config *helm.Configuration) error {
	pass := i.checker.Run(check.PreinstallChecks(), func(r *check.Result) {
		if r.Err != nil {
			fmt.Fprintln(i.out, "ERROR:", r.Err)
		}
	})
	if !pass {
		return errors.New("Pre-install checks failed")
	}

	var chartRequested *chart.Chart
	var err error
	if i.chartPath != "" {
		chartRequested, err = loader.Load(i.chartPath)
	} else {
		chartRequested, err = cli.LoadChart(chartTGZSource)
	}
	if err != nil {
		return err
	}

	meshNameErrs := validation.IsDNS1123Label(i.meshName)

	if len(meshNameErrs) != 0 {
		return fmt.Errorf("Invalid mesh-name: %v", meshNameErrs)
	}

	if strings.EqualFold(i.certManager, "vault") {
		var missingFields []string
		if i.vaultHost == "" {
			missingFields = append(missingFields, "vault-host")
		}
		if i.vaultToken == "" {
			missingFields = append(missingFields, "vault-token")
		}
		if len(missingFields) != 0 {
			return fmt.Errorf("Missing arguments for cert-manager vault: %v", missingFields)
		}
	}

	// Validate CIDR ranges if egress is enabled
	if i.enableEgress {
		if err := validateCIDRs(i.meshCIDRRanges); err != nil {
			return fmt.Errorf("Invalid mesh-cidr-ranges: %q, error: %v", i.meshCIDRRanges, err)
		}
	}

	values, err := i.resolveValues()
	if err != nil {
		return err
	}

	listClient := helm.NewList(config)
	listClient.AllNamespaces = true
	releases, err := listClient.Run()
	if err != nil {
		return err
	}
	for _, release := range releases {
		if osmVals, exists := release.Config["OpenServiceMesh"]; exists {
			if valsMap, ok := osmVals.(map[string]interface{}); ok {
				if meshName, exists := valsMap["meshName"]; exists && meshName == i.meshName {
					return errMeshAlreadyExists(i.meshName)
				}
			}
		}
	}

	installClient := helm.NewInstall(config)
	installClient.ReleaseName = i.meshName
	installClient.Namespace = settings.Namespace()
	installClient.CreateNamespace = true
	if _, err = installClient.Run(chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(i.out, "OSM installed successfully in namespace [%s] with mesh name [%s]\n", settings.Namespace(), i.meshName)
	return nil
}

func (i *installCmd) resolveValues() (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}
	valuesConfig := []string{
		fmt.Sprintf("OpenServiceMesh.image.registry=%s", i.containerRegistry),
		fmt.Sprintf("OpenServiceMesh.image.tag=%s", i.osmImageTag),
		fmt.Sprintf("OpenServiceMesh.imagePullSecrets[0].name=%s", i.containerRegistrySecret),
		fmt.Sprintf("OpenServiceMesh.certManager=%s", i.certManager),
		fmt.Sprintf("OpenServiceMesh.vault.host=%s", i.vaultHost),
		fmt.Sprintf("OpenServiceMesh.vault.protocol=%s", i.vaultProtocol),
		fmt.Sprintf("OpenServiceMesh.vault.token=%s", i.vaultToken),
		fmt.Sprintf("OpenServiceMesh.vault.role=%s", i.vaultRole),
		fmt.Sprintf("OpenServiceMesh.serviceCertValidityMinutes=%d", i.serviceCertValidityMinutes),
		fmt.Sprintf("OpenServiceMesh.prometheus.retention.time=%s", i.prometheusRetentionTime),
		fmt.Sprintf("OpenServiceMesh.enableDebugServer=%t", i.enableDebugServer),
		fmt.Sprintf("OpenServiceMesh.enablePermissiveTrafficPolicy=%t", i.enablePermissiveTrafficPolicy),
		fmt.Sprintf("OpenServiceMesh.enableBackpressureExperimental=%t", i.enableBackpressureExperimental),
		fmt.Sprintf("OpenServiceMesh.enableMetricsStack=%t", i.enableMetricsStack),
		fmt.Sprintf("OpenServiceMesh.meshName=%s", i.meshName),
		fmt.Sprintf("OpenServiceMesh.enableEgress=%t", i.enableEgress),
		fmt.Sprintf("OpenServiceMesh.meshCIDRRanges=%s", strings.Join(i.meshCIDRRanges, " ")),
	}

	for _, val := range valuesConfig {
		if err := strvals.ParseInto(val, finalValues); err != nil {
			return finalValues, err
		}
	}
	return finalValues, nil
}

func errMeshAlreadyExists(name string) error {
	return fmt.Errorf("Mesh %s already exists in cluster. Please specify a new mesh name using --mesh-name", name)
}

func validateCIDRs(cidrRanges []string) error {
	if len(cidrRanges) == 0 {
		return fmt.Errorf("CIDR ranges cannot be empty")
	}
	for _, cidr := range cidrRanges {
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("Error parsing CIDR %s", cidr)
		}
	}
	return nil
}

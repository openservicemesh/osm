package main

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/openservicemesh/osm/pkg/cli"
)

const (
	defaultUseHTTPSIngress = false
	defaultEnableTracing   = true
)

const upgradeDesc = `
This command upgrades an OSM control plane configuration by upgrading the
underlying Helm release.

The mesh to upgrade is identified by its mesh name and namespace. If either were
overridden from the default for the "osm install" command, the --mesh-name and
--osm-namespace flags need to be specified.

By default, if flags tied to Helm chart values are not specified for "osm mesh
upgrade", then their current value will be carried over to the new release. Two
exceptions to this rule are --container-registry and --osm-image-tag, which
will be overridden from the old release by default.

If any CustomResourceDefinitions (CRDs) are different between the installed
chart and the upgraded chart, the CRDs (and any corresponding custom resources)
need to be deleted and recreated using the CRDs in the new chart prior to
updating the mesh to ensure compatibility.
`

const meshUpgradeExample = `
# Upgrade the mesh with the default name in the osm-system namespace, setting
# OpenServiceMesh.enableEgress to false, setting the image registry and tag to
# the defaults, and leaving all other values unchanged.
osm mesh upgrade --osm-namespace osm-system --enable-egress=false
`

type meshUpgradeCmd struct {
	out io.Writer

	meshName  string
	chartPath string

	containerRegistry string
	osmImageTag       string

	// Bools are pointers so we can differentiate between true/false/unset
	enablePermissiveTrafficPolicy *bool
	enableEgress                  *bool
	enableDebugServer             *bool
	envoyLogLevel                 string
	enablePrometheusScraping      *bool
	useHTTPSIngress               *bool
	serviceCertValidityDuration   time.Duration
	enableTracing                 *bool
	tracingAddress                string
	tracingPort                   uint16
	tracingEndpoint               string
}

func newMeshUpgradeCmd(config *helm.Configuration, out io.Writer) *cobra.Command {
	upg := &meshUpgradeCmd{
		out: out,

		// Bool pointers need to be non-nil before pflag can use them to store
		// values.
		enableEgress:                  new(bool),
		enablePermissiveTrafficPolicy: new(bool),
		enableDebugServer:             new(bool),
		enablePrometheusScraping:      new(bool),
		useHTTPSIngress:               new(bool),
		enableTracing:                 new(bool),
	}

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "upgrade osm control plane configuration",
		Long:    upgradeDesc,
		Example: meshUpgradeExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// By default, bool flags should remain unchanged unless explicitly set or unset.
			f := cmd.Flags()
			if !f.Changed("enable-egress") {
				upg.enableEgress = nil
			}
			if !f.Changed("enable-permissive-traffic-policy") {
				upg.enablePermissiveTrafficPolicy = nil
			}
			if !f.Changed("enable-debug-server") {
				upg.enableDebugServer = nil
			}
			if !f.Changed("enable-prometheus-scraping") {
				upg.enablePrometheusScraping = nil
			}
			if !f.Changed("use-https-ingress") {
				upg.useHTTPSIngress = nil
			}
			if !f.Changed("enable-tracing") {
				upg.enableTracing = nil
			}

			return upg.run(config)
		},
	}

	f := cmd.Flags()

	f.StringVar(&upg.meshName, "mesh-name", defaultMeshName, "Name of the mesh to upgrade")
	f.StringVar(&upg.chartPath, "osm-chart-path", "", "path to osm chart to override default chart")
	f.StringVar(&upg.containerRegistry, "container-registry", defaultContainerRegistry, "container registry that hosts control plane component images")
	f.StringVar(&upg.osmImageTag, "osm-image-tag", defaultOsmImageTag, "osm image tag")

	f.BoolVar(upg.enablePermissiveTrafficPolicy, "enable-permissive-traffic-policy", defaultEnablePermissiveTrafficPolicy, "Enable permissive traffic policy mode")
	f.BoolVar(upg.enableEgress, "enable-egress", defaultEnableEgress, "Enable egress in the mesh")
	f.BoolVar(upg.enableDebugServer, "enable-debug-server", defaultEnableDebugServer, "Enable the debug HTTP server")
	f.StringVar(&upg.envoyLogLevel, "envoy-log-level", "", "Envoy log level is used to specify the level of logs collected from envoy and needs to be one of these (trace, debug, info, warning, warn, error, critical, off)")
	f.BoolVar(upg.enablePrometheusScraping, "enable-prometheus-scraping", defaultEnablePrometheusScraping, "Enable Prometheus metrics scraping on sidecar proxies")
	f.BoolVar(upg.useHTTPSIngress, "use-https-ingress", defaultUseHTTPSIngress, "Enable HTTPS Ingress")
	f.DurationVar(&upg.serviceCertValidityDuration, "service-cert-validity-duration", 0, "Service certificate validity duration, represented as a sequence of decimal numbers each with optional fraction and a unit suffix")
	f.BoolVar(upg.enableTracing, "enable-tracing", defaultEnableTracing, "Enable tracing")
	f.StringVar(&upg.tracingAddress, "tracing-address", "", "Tracing server hostname")
	f.Uint16Var(&upg.tracingPort, "tracing-port", 0, "Tracing server port")
	f.StringVar(&upg.tracingEndpoint, "tracing-endpoint", "", "Tracing server endpoint")

	return cmd
}

func (u *meshUpgradeCmd) run(config *helm.Configuration) error {
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

	// Add the overlay values to be updated to the current release's values map
	values, err := u.resolveValues()
	if err != nil {
		return err
	}

	upgradeClient := helm.NewUpgrade(config)
	upgradeClient.Wait = true
	upgradeClient.ReuseValues = true
	if _, err = upgradeClient.Run(u.meshName, chartRequested, values); err != nil {
		return err
	}

	fmt.Fprintf(u.out, "OSM successfully upgraded mesh %s\n", u.meshName)
	return nil
}

func (u *meshUpgradeCmd) resolveValues() (map[string]interface{}, error) {
	vals := map[string]interface{}{
		"image": map[string]interface{}{
			"tag":      u.osmImageTag,
			"registry": u.containerRegistry,
		},
	}

	if u.enablePermissiveTrafficPolicy != nil {
		vals["enablePermissiveTrafficPolicy"] = *u.enablePermissiveTrafficPolicy
	}
	if u.enableEgress != nil {
		vals["enableEgress"] = *u.enableEgress
	}
	if u.enableDebugServer != nil {
		vals["enableDebugServer"] = *u.enableDebugServer
	}
	if len(u.envoyLogLevel) > 0 {
		vals["envoyLogLevel"] = u.envoyLogLevel
	}
	if u.enablePrometheusScraping != nil {
		vals["enablePrometheusScraping"] = *u.enablePrometheusScraping
	}
	if u.useHTTPSIngress != nil {
		vals["useHTTPSIngress"] = *u.useHTTPSIngress
	}
	if u.serviceCertValidityDuration > 0 {
		vals["serviceCertValidityDuration"] = u.serviceCertValidityDuration
	}

	setTracing := func(key string, val interface{}) {
		if _, exists := vals["tracing"]; !exists {
			vals["tracing"] = map[string]interface{}{}
		}
		vals["tracing"].(map[string]interface{})[key] = val
	}
	if u.enableTracing != nil {
		setTracing("enable", *u.enableTracing)
	}
	if len(u.tracingAddress) > 0 {
		setTracing("address", u.tracingAddress)
	}
	if u.tracingPort > 0 {
		setTracing("port", u.tracingPort)
	}
	if len(u.tracingEndpoint) > 0 {
		setTracing("endpoint", u.tracingEndpoint)
	}

	return map[string]interface{}{
		"OpenServiceMesh": vals,
	}, nil
}

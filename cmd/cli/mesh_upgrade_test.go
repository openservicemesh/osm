package main

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func meshUpgradeConfig() *action.Configuration {
	mem := driver.NewMemory()
	mem.SetNamespace(settings.Namespace())
	store := storage.Init(mem)

	return &action.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(_ string, _ ...interface{}) {},
	}
}

func defaultMeshUpgradeCmd() *meshUpgradeCmd {
	chart, err := loader.Load(testChartPath)
	if err != nil {
		panic(err)
	}

	return &meshUpgradeCmd{
		out:               ioutil.Discard,
		meshName:          defaultMeshName,
		chart:             chart,
		containerRegistry: defaultContainerRegistry,
		osmImageTag:       defaultOsmImageTag,
	}
}

func TestMeshUpgradeDefault(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	upgraded, err := action.NewGet(config).Run(u.meshName)
	a.Nil(err)

	osmImageTag, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.image.tag")
	a.Nil(err)
	a.Equal(defaultOsmImageTag, osmImageTag)

	err = u.run(config)
	a.Nil(err)
}

func TestMeshUpgradeOverridesInstallDefaults(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()
	u.osmImageTag = "upgraded"
	u.enableEgress = new(bool)
	*u.enableEgress = true
	u.envoyLogLevel = "trace"
	u.tracingEndpoint = "/here"

	err = u.run(config)
	a.Nil(err)

	upgraded, err := action.NewGet(config).Run(u.meshName)
	a.Nil(err)

	osmImageTag, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.image.tag")
	a.Nil(err)
	a.Equal("upgraded", osmImageTag)

	egressEnabled, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.enableEgress")
	a.Nil(err)
	a.True(egressEnabled.(bool))

	envoyLogLevel, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.envoyLogLevel")
	a.Nil(err)
	a.Equal("trace", envoyLogLevel)

	tracingEndpoint, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.tracing.endpoint")
	a.Nil(err)
	a.Equal("/here", tracingEndpoint)

	// Successive upgrades should keep the overridden values from the previous upgrade
	u = defaultMeshUpgradeCmd()
	err = u.run(config)
	a.Nil(err)

	upgraded, err = action.NewGet(config).Run(u.meshName)
	a.Nil(err)

	tracingEndpoint, err = chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.tracing.endpoint")
	a.Nil(err)
	a.Equal("/here", tracingEndpoint)
}

func TestMeshUpgradeKeepsInstallOverrides(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath
	i.enableEgress = !defaultEnableEgress
	i.osmImageTag = "installed"
	i.envoyLogLevel = "trace"

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	err = u.run(config)
	a.Nil(err)

	upgraded, err := action.NewGet(config).Run(u.meshName)
	a.Nil(err)

	// enableEgress should be unchanged by default
	egressEnabled, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.enableEgress")
	a.Nil(err)
	a.Equal(!defaultEnableEgress, egressEnabled)

	// envoyLogLevel should be unchanged by default
	envoyLogLevel, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.envoyLogLevel")
	a.Nil(err)
	a.Equal("trace", envoyLogLevel)

	// image tag should be updated by default
	tag, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.image.tag")
	a.Nil(err)
	a.Equal(defaultOsmImageTag, tag)
}

func TestMeshUpgradeToModifiedChart(t *testing.T) {
	a := assert.New(t)

	config := meshUpgradeConfig()

	i := getDefaultInstallCmd(ioutil.Discard)
	i.chartPath = testChartPath

	err := i.run(config)
	a.Nil(err)

	u := defaultMeshUpgradeCmd()

	// Upgrade to a "new version" of the chart with a new required value.
	// The upgrade itself will fail if the required value is not defined.
	_, exists := u.chart.Values["newRequired"]
	a.False(exists)
	u.chart.Values["newRequired"] = "anything"
	u.chart.Schema = []byte(`{"required": ["newRequired"]}`)

	// A value NOT set explicitly by `osm install` that exists in the old chart
	oldNamespace, err := chartutil.Values(u.chart.Values).PathValue("OpenServiceMesh.namespace")
	a.Nil(err)
	newNamespace := fmt.Sprintf("new-%s", oldNamespace)
	u.chart.Values["OpenServiceMesh"].(map[string]interface{})["namespace"] = newNamespace

	err = u.run(config)
	a.Nil(err)

	upgraded, err := action.NewGet(config).Run(u.meshName)
	a.Nil(err)

	// When a default is changed in values.yaml, keep the old one
	namespace, err := chartutil.Values(upgraded.Config).PathValue("OpenServiceMesh.namespace")
	a.Nil(err)
	a.Equal(oldNamespace, namespace)
}

package main

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func meshUpgradeConfig() *helm.Configuration {
	mem := driver.NewMemory()
	mem.SetNamespace(settings.Namespace())
	store := storage.Init(mem)

	return &helm.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard,
		},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(_ string, _ ...interface{}) {},
	}
}

func defaultMeshUpgradeCmd() *meshUpgradeCmd {
	return &meshUpgradeCmd{
		out:               ioutil.Discard,
		meshName:          defaultMeshName,
		chartPath:         testChartPath,
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

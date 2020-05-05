package main

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	helm "helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

var (
	testRegistry       = "test-registry"
	testRegistrySecret = "test-registry-secret"
)

func TestInstallRun(t *testing.T) {
	out := new(bytes.Buffer)
	store := storage.Init(driver.NewMemory())
	if mem, ok := store.Driver.(*driver.Memory); ok {
		mem.SetNamespace(settings.Namespace())
	}

	config := &helm.Configuration{
		Releases: store,
		KubeClient: &kubefake.PrintingKubeClient{
			Out: ioutil.Discard},
		Capabilities: chartutil.DefaultCapabilities,
		Log:          func(format string, v ...interface{}) {},
	}

	installCmd := &installCmd{
		out:                     out,
		chartPath:               "testdata/test-chart",
		containerRegistry:       testRegistry,
		containerRegistrySecret: testRegistrySecret,
	}

	installClient := helm.NewInstall(config)
	if err := installCmd.run(installClient); err != nil {
		t.Fatal(err)
	}

	expectedOutput := "OSM installed successfully in osm-system namespace\n"
	result := out.String()
	if result != expectedOutput {
		t.Errorf("Expected %s, got %s", expectedOutput, result)
	}

	rel, err := config.Releases.Get(defaultHelmReleaseName, 1)
	if err != nil {
		t.Errorf("Expected helm release %s, got err %s", defaultHelmReleaseName, err)
	}

	//user did not set any values. Used same defaults from test-chart so this is empty.
	expectedUserValues := map[string]interface{}{
		"image": map[string]interface{}{
			"registry": testRegistry,
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": testRegistrySecret,
			},
		},
		"namespace": settings.Namespace(),
	}
	if !reflect.DeepEqual(rel.Config, expectedUserValues) {
		t.Errorf("Expected helm release values to resolve as %#v\nbut got %#v", expectedUserValues, rel.Config)
	}

	if rel.Namespace != settings.Namespace() {
		t.Errorf("Expected helm release namespace to be %s, got %s", settings.Namespace(), rel.Namespace)
	}

}

func TestResolveValues(t *testing.T) {

	installCmd := &installCmd{
		containerRegistry:       testRegistry,
		containerRegistrySecret: testRegistrySecret,
	}

	vals, err := installCmd.resolveValues()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{
		"image": map[string]interface{}{
			"registry": testRegistry,
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": testRegistrySecret,
			},
		},
		"namespace": settings.Namespace(),
	}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("Expected values to resolve as %#v\nbut got %#v", expected, vals)
	}
}

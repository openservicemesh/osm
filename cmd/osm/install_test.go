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
	testVaultHost      = "vault.osm.svc.cluster.local"
	testVaultProtocol  = "http"
	testVaultToken     = "token"
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
		osmID:                   "testid-123",
		certManager:             "tresor",
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

	rel, err := config.Releases.Get(installCmd.osmID, 1)
	if err != nil {
		t.Errorf("Expected helm release %s, got err %s", installCmd.osmID, err)
	}

	//user did not set any values. Used same defaults from test-chart so this is empty.
	expectedUserValues := map[string]interface{}{
		"certManager": "tresor",
		"image": map[string]interface{}{
			"registry": testRegistry,
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": testRegistrySecret,
			},
		},
		"namespace": settings.Namespace(),
		"osmID":     installCmd.osmID,
		"vault": map[string]interface{}{
			"host":     "",
			"protocol": "",
			"token":    "",
		},
	}
	if !reflect.DeepEqual(rel.Config, expectedUserValues) {
		t.Errorf("Expected helm release values to resolve as %#v\nbut got %#v", expectedUserValues, rel.Config)
	}

	if rel.Namespace != settings.Namespace() {
		t.Errorf("Expected helm release namespace to be %s, got %s", settings.Namespace(), rel.Namespace)
	}
}

func TestInstallRunVault(t *testing.T) {
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
		certManager:             "vault",
		vaultHost:               testVaultHost,
		vaultToken:              testVaultToken,
		vaultProtocol:           "http",
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

	rel, err := config.Releases.Get(installCmd.osmID, 1)
	if err != nil {
		t.Errorf("Expected helm release %s, got err %s", installCmd.osmID, err)
	}

	expectedUserValues := map[string]interface{}{
		"certManager": "vault",
		"image": map[string]interface{}{
			"registry": testRegistry,
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": testRegistrySecret,
			},
		},
		"namespace": settings.Namespace(),
		"osmID":     installCmd.osmID,
		"vault": map[string]interface{}{
			"host":     testVaultHost,
			"protocol": "http",
			"token":    testVaultToken,
		},
	}
	if !reflect.DeepEqual(rel.Config, expectedUserValues) {
		t.Errorf("Expected helm release values to resolve as %#v\nbut got %#v", expectedUserValues, rel.Config)
	}

	if rel.Namespace != settings.Namespace() {
		t.Errorf("Expected helm release namespace to be %s, got %s", settings.Namespace(), rel.Namespace)
	}
}

func TestInstallRunVaultNoArgs(t *testing.T) {
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
		certManager:             "vault",
	}

	installClient := helm.NewInstall(config)
	err := installCmd.run(installClient)
	expectedError := "Missing arguments for cert-manager vault: [vault-host vault-token]"
	if err == nil {
		t.Errorf("No error occurred. Expected error: %s", expectedError)
	} else if expectedError != err.Error() {
		t.Errorf("Expected error: (%s) but got (%s)", expectedError, err.Error())
	}
}
func TestResolveValues(t *testing.T) {

	installCmd := &installCmd{
		containerRegistry:       testRegistry,
		containerRegistrySecret: testRegistrySecret,
		osmID:                   "helloworld",
		certManager:             "vault",
		vaultHost:               testVaultHost,
		vaultProtocol:           testVaultProtocol,
		vaultToken:              testVaultToken,
	}

	vals, err := installCmd.resolveValues()
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]interface{}{
		"certManager": "vault",
		"image": map[string]interface{}{
			"registry": testRegistry,
		},
		"imagePullSecrets": []interface{}{
			map[string]interface{}{
				"name": testRegistrySecret,
			},
		},
		"namespace": settings.Namespace(),
		"osmID":     installCmd.osmID,
		"vault": map[string]interface{}{
			"host":     testVaultHost,
			"protocol": "http",
			"token":    testVaultToken,
		},
	}
	if !reflect.DeepEqual(vals, expected) {
		t.Errorf("Expected values to resolve as %#v\nbut got %#v", expected, vals)
	}
}

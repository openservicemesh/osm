package main

import (
	"context"
	"testing"

	"github.com/naoina/toml"
	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/fake"
)

const (
	testOsmNamespace = "testNamespace"
)

func TestMetricsAddons(t *testing.T) {
	assert := tassert.New(t)
	fakeClient := fake.NewSimpleClientset()

	// Test temp variables
	var azmonSettings azMonConfStruct

	_, err := fakeClient.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testOsmNamespace,
		},
	}, metav1.CreateOptions{})
	assert.NoError(err)

	// Lack of common cmd ancestor made hard to do this through test waterfall

	// With no config map, disable does nothing
	err = (&metricsDisableCmd{
		namespaces:  []string{"foo", "bar", "cat"},
		azmonConfig: true,
		clientSet:   fakeClient,
	}).runAzmonDisable(testOsmNamespace)
	assert.NoError(err)

	// Verify nothing was created
	_, err = fakeClient.CoreV1().ConfigMaps(testOsmNamespace).Get(context.Background(), azmonConfigMapName, metav1.GetOptions{})
	assert.Error(err)

	// Call enable for Az mon
	err = (&metricsEnableCmd{
		namespaces:  []string{"foo", "bar", "cat"},
		azmonConfig: true,
		clientSet:   fakeClient,
	}).runAzmonEnable(testOsmNamespace)
	assert.NoError(err)

	// Verify config map was created
	cfgMap, err := fakeClient.CoreV1().ConfigMaps(testOsmNamespace).Get(context.Background(), azmonConfigMapName, metav1.GetOptions{})
	assert.NoError(err)

	// Verify Configmap schema/version
	assert.Equal(supportedSchema, cfgMap.Data[cfgMapSchemaVersionKey])
	assert.Equal("1", cfgMap.Data[cfgMapConfigVersionKey])

	// Verify TOML in configmap
	assert.NoError(toml.Unmarshal([]byte(cfgMap.Data[azmonCfmapOsmKey]), &azmonSettings))
	for _, testNs := range []string{"foo", "bar", "cat"} {
		assert.Contains(azmonSettings.AzMonCollectionConf.Settings.MonitorNs, testNs)
	}

	// Call disable on a delta only, should just leave bar
	err = (&metricsDisableCmd{
		namespaces:  []string{"foo", "cat"},
		azmonConfig: true,
		clientSet:   fakeClient,
	}).runAzmonDisable(testOsmNamespace)
	assert.NoError(err)

	// Config map will exist, will be empty in TOML namespace content
	cfgMap, err = fakeClient.CoreV1().ConfigMaps(testOsmNamespace).Get(context.Background(), azmonConfigMapName, metav1.GetOptions{})
	assert.NoError(err)

	// Verify Configmap schema/version
	assert.Equal(supportedSchema, cfgMap.Data[cfgMapSchemaVersionKey])
	assert.Equal("2", cfgMap.Data[cfgMapConfigVersionKey])

	// Verify TOML in configmap
	assert.NoError(toml.Unmarshal([]byte(cfgMap.Data[azmonCfmapOsmKey]), &azmonSettings))
	assert.Equal([]string{"bar"}, azmonSettings.AzMonCollectionConf.Settings.MonitorNs)
}

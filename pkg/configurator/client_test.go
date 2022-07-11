package configurator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
	"github.com/openservicemesh/osm/pkg/k8s/informers"
	"github.com/openservicemesh/osm/pkg/metricsstore"

	configv1alpha2 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha2"
)

const (
	osmNamespace      = "-test-osm-namespace-"
	osmMeshConfigName = "-test-osm-mesh-config-"
)

func TestGetMeshConfig(t *testing.T) {
	a := assert.New(t)

	meshConfigClient := fakeConfig.NewSimpleClientset()
	stop := make(chan struct{})

	ic, err := informers.NewInformerCollection("osm", stop, informers.WithConfigClient(meshConfigClient, osmMeshConfigName, osmNamespace))
	a.Nil(err)

	c := NewConfigurator(ic, osmNamespace, osmMeshConfigName, nil)

	// Returns empty MeshConfig if informer cache is empty
	a.Equal(configv1alpha2.MeshConfig{}, c.getMeshConfig())

	newObj := &configv1alpha2.MeshConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.openservicemesh.io",
			Kind:       "MeshConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: osmNamespace,
			Name:      osmMeshConfigName,
		},
	}
	err = c.informers.Add(informers.InformerKeyMeshConfig, newObj, t)
	a.Nil(err)
	a.Equal(*newObj, c.getMeshConfig())
}

func TestMetricsHandler(t *testing.T) {
	a := assert.New(t)

	c := &Client{
		meshConfigName: osmMeshConfigName,
		informers:      &informers.InformerCollection{},
	}
	handlers := c.metricsHandler()
	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.FeatureFlagEnabled)

	// Adding the MeshConfig
	handlers.OnAdd(&configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableRetryPolicy: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 1` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))

	// Updating the MeshConfig
	handlers.OnUpdate(nil, &configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 1` + "\n"))

	// Deleting the MeshConfig
	handlers.OnDelete(&configv1alpha2.MeshConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: osmMeshConfigName,
		},
		Spec: configv1alpha2.MeshConfigSpec{
			FeatureFlags: configv1alpha2.FeatureFlags{
				EnableSnapshotCacheMode: true,
			},
		},
	})
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n"))
	a.True(metricsstore.DefaultMetricsStore.Contains(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n"))
}

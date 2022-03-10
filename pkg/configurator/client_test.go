package configurator

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
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
	c := newConfigurator(meshConfigClient, stop, osmNamespace, osmMeshConfigName, nil)

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
	err := c.cache.Add(newObj)
	a.Nil(err)
	a.Equal(*newObj, c.getMeshConfig())
}

type store struct {
	cache.Store
}

func (s *store) GetByKey(_ string) (interface{}, bool, error) {
	return nil, false, nil
}

func TestMetricsHandler(t *testing.T) {
	c := &client{
		cache:          &store{},
		meshConfigName: osmMeshConfigName,
	}
	handlers := c.metricsHandler()
	metricsstore.DefaultMetricsStore.Start(metricsstore.DefaultMetricsStore.FeatureFlagEnabled)

	assertMetricsContain := func(metric string) {
		t.Helper()
		req := httptest.NewRequest("GET", "http://this.doesnt/matter", nil)
		w := httptest.NewRecorder()
		metricsstore.DefaultMetricsStore.Handler().ServeHTTP(w, req)
		res := w.Body.String()

		assert.Contains(t, res, metric)
	}

	// Adding a MeshConfig
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
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 1` + "\n")
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n")

	// Updating the "real" MeshConfig
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
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n")
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 1` + "\n")

	// Deleting the "real" MeshConfig
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
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableRetryPolicy"} 0` + "\n")
	assertMetricsContain(`osm_feature_flag_enabled{feature_flag="enableSnapshotCacheMode"} 0` + "\n")
}

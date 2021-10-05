package configurator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fakeConfig "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
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
	a.Equal(&v1alpha1.MeshConfig{}, c.getMeshConfig())

	newObj := &v1alpha1.MeshConfig{
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
	a.Equal(newObj, c.getMeshConfig())
}

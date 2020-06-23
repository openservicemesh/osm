package configurator

import (
	v1 "github.com/open-service-mesh/osm/pkg/apis/osmconfig/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// FakeConfigurator is a fake namespace.Configurator used for testing
type FakeConfigurator struct {
	namespaces []string
	Configurator
}

type fakeCache struct {
	namespaces []string
	ingresses  []string
}

// GetByKey returns the accumulator associated with the given key
func (c fakeCache) GetByKey(key string) (item interface{}, exists bool, err error) {
	return &v1.OSMConfig{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.OSMConfigSpec{
			TypeMeta:     metav1.TypeMeta{},
			ObjectMeta:   metav1.ObjectMeta{},
			LogVerbosity: "",
			Namespaces:   c.namespaces,
			Ingresses:    c.ingresses,
		},
	}, true, nil
}

// Add adds the given object to the accumulator associated with the given object's key
func (c fakeCache) Add(obj interface{}) error { return nil }

// Update updates the given object in the accumulator associated with the given object's key
func (c fakeCache) Update(obj interface{}) error { return nil }

// Delete deletes the given object from the accumulator associated with the given object's key
func (c fakeCache) Delete(obj interface{}) error { return nil }

// List returns a list of all the currently non-empty accumulators
func (c fakeCache) List() []interface{} { return nil }

// ListKeys returns a list of all the keys currently associated with non-empty accumulators
func (c fakeCache) ListKeys() []string { return nil }

// Get returns the accumulator associated with the given object's key
func (c fakeCache) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Replace will delete the contents of the store, using instead the
// given list. Store takes ownership of the list, you should not reference
// it after calling this function.
func (c fakeCache) Replace([]interface{}, string) error { return nil }

// Resync is meaningless in the terms appearing here but has
// meaning in some implementations that have non-trivial
// additional behavior (e.g., DeltaFIFO).
func (c fakeCache) Resync() error { return nil }

// NewFakeConfigurator creates a fake configurator.
func NewFakeConfigurator(namespaces []string, kubeclient kubernetes.Interface) Client {
	client := Client{
		configCRDName:      "test-osm-config",
		configCRDNamespace: "test-osm-namespace",
		informer:           nil,
		cache: fakeCache{
			namespaces: namespaces,
			ingresses:  []string{},
		},
		cacheSynced:   make(chan interface{}),
		announcements: make(chan interface{}),
	}
	return client
}

// IsMonitoredNamespace returns if the namespace is monitored
func (f FakeConfigurator) IsMonitoredNamespace(namespace string) bool {
	log.Debug().Msgf("Monitored namespaces = %v", f.namespaces)
	for _, ns := range f.namespaces {
		if ns == namespace {
			return true
		}
	}
	return false
}

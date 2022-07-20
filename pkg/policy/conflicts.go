package policy

import (
	"fmt"

	mapset "github.com/deckarep/golang-set"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// DetectIngressBackendConflicts detects conflicts between the given IngressBackend resources
func DetectIngressBackendConflicts(x policyv1alpha1.IngressBackend, y policyv1alpha1.IngressBackend) []error {
	var conflicts []error // multiple conflicts could exist

	// Check if the backends conflict
	xSet := mapset.NewSet()
	type setKey struct {
		name string
		port int
	}
	for _, backend := range x.Spec.Backends {
		key := setKey{
			name: backend.Name,
			port: backend.Port.Number,
		}
		xSet.Add(key)
	}
	ySet := mapset.NewSet()
	for _, backend := range y.Spec.Backends {
		key := setKey{
			name: backend.Name,
			port: backend.Port.Number,
		}
		ySet.Add(key)
	}

	duplicates := xSet.Intersect(ySet)
	for b := range duplicates.Iter() {
		err := fmt.Errorf("Backend %s specified in %s and %s conflicts", b.(setKey).name, x.Name, y.Name)
		conflicts = append(conflicts, err)
	}

	return conflicts
}

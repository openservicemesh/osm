package policy

import (
	mapset "github.com/deckarep/golang-set"
	"github.com/pkg/errors"

	policyv1alpha1 "github.com/openservicemesh/osm/pkg/apis/policy/v1alpha1"
)

// DetectIngressBackendConflicts detects conflicts between the given IngressBackend resources
func DetectIngressBackendConflicts(x policyv1alpha1.IngressBackend, y policyv1alpha1.IngressBackend) []error {
	var conflicts []error // multiple conflicts could exist

	// Check if the backends conflict
	xSet := mapset.NewSet()
	for _, backend := range x.Spec.Backends {
		xSet.Add(backend.Name)
	}
	ySet := mapset.NewSet()
	for _, backend := range y.Spec.Backends {
		ySet.Add(backend.Name)
	}

	duplicates := xSet.Intersect(ySet)
	for b := range duplicates.Iter() {
		err := errors.Errorf("Backend %s specified in %s and %s conflicts", b.(string), x.Name, y.Name)
		conflicts = append(conflicts, err)
	}

	return conflicts
}

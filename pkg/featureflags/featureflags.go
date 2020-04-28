package featureflags

import (
	"sync"

	"github.com/open-service-mesh/osm/pkg/logger"
)

// OptionalFeatures is a struct to enable/disable optional features
type OptionalFeatures struct {
	Ingress bool
}

var (
	// Features describes whether an optional feature is enabled
	Features OptionalFeatures

	once sync.Once
	log  = logger.New("featureflags")
)

// Initialize initializes the feature flag options
func Initialize(optionalFeatures OptionalFeatures) {
	once.Do(func() {
		Features = optionalFeatures
	})
}

// IsIngressEnabled returns a boolean indicating if the ingress feature is enabled
func IsIngressEnabled() bool {
	return Features.Ingress
}

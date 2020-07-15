package featureflags

import (
	"sync"
)

// OptionalFeatures is a struct to enable/disable optional features
type OptionalFeatures struct {
	// FeatureName bool
	Backpressure bool

	// Enables / disables colorful and pretty logging.
	// This is not expected to change during the lifetime of the OSM controller pod.
	EnableHumanReadableLog bool
}

var (
	// Features describes whether an optional feature is enabled
	Features OptionalFeatures

	once sync.Once
)

// Initialize initializes the feature flag options
func Initialize(optionalFeatures OptionalFeatures) {
	once.Do(func() {
		Features = optionalFeatures
	})
}

/* Feature flag stub
// IsFeatureNameEnabled returns a boolean indicating if the feature `FeatureName` is enabled
func IsFeatureNameEnabled() bool {
	return Features.FeatureName
}
*/

// IsBackpressureEnabled returns a boolean indicating if the experimental backpressure feature is enabled
func IsBackpressureEnabled() bool {
	return Features.Backpressure
}

func EnableHumanReadalbeLog() bool {
	return Features.EnableHumanReadableLog
}

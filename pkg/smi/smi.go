package smi

import (
	smiAccess "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiSplit "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
)

// FilterTrafficSplit applies the given TrafficSplitListOption filter on the given TrafficSplit object
func FilterTrafficSplit(trafficSplit *smiSplit.TrafficSplit, options ...TrafficSplitListOption) *smiSplit.TrafficSplit {
	if trafficSplit == nil {
		return nil
	}

	o := &TrafficSplitListOpt{}
	for _, opt := range options {
		opt(o)
	}

	// If apex service filter option is set, ignore traffic splits whose apex service does not match
	if o.ApexService.Name != "" && (o.ApexService.Namespace != trafficSplit.Namespace ||
		o.ApexService.Name != trafficSplit.Spec.Service) {
		return nil
	}

	// If backend service filter option is set, ignore traffic splits whose backend service does not match
	if o.BackendService.Name != "" {
		if trafficSplit.Namespace != o.BackendService.Namespace {
			return nil
		}

		backendFound := false
		for _, backend := range trafficSplit.Spec.Backends {
			if backend.Service == o.BackendService.Name {
				backendFound = true
				break
			}
		}
		if !backendFound {
			return nil
		}
	}

	return trafficSplit
}

// FilterTrafficTarget applies the given TrafficTargetListOption filter on the given TrafficTarget object
func FilterTrafficTarget(trafficTarget *smiAccess.TrafficTarget, options ...TrafficTargetListOption) *smiAccess.TrafficTarget {
	if trafficTarget == nil {
		return nil
	}

	o := &TrafficTargetListOpt{}
	for _, opt := range options {
		opt(o)
	}

	if o.Destination.Name != "" && (o.Destination.Namespace != trafficTarget.Spec.Destination.Namespace ||
		o.Destination.Name != trafficTarget.Spec.Destination.Name) {
		return nil
	}

	return trafficTarget
}

// IsValidTrafficTarget checks if the given SMI TrafficTarget object is valid
func IsValidTrafficTarget(trafficTarget *smiAccess.TrafficTarget) bool {
	// destination namespace must be same as traffic target namespace
	if trafficTarget.Namespace != trafficTarget.Spec.Destination.Namespace {
		return false
	}

	if !HasValidRules(trafficTarget.Spec.Rules) {
		return false
	}

	return true
}

// HasValidRules checks if the given SMI TrafficTarget object has valid rules
func HasValidRules(rules []smiAccess.TrafficTargetRule) bool {
	if len(rules) == 0 {
		return false
	}
	for _, rule := range rules {
		switch rule.Kind {
		case HTTPRouteGroupKind, TCPRouteKind:
			// valid Kind for rules

		default:
			log.Error().Msgf("Invalid Kind for rule %s in TrafficTarget policy %s", rule.Name, rule.Kind)
			return false
		}
	}
	return true
}

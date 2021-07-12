package trafficpolicy

// IngressTrafficPolicy defines the ingress traffic match and routes for a given backend
type IngressTrafficPolicy struct {
	TrafficMatches    []*IngressTrafficMatch
	HTTPRoutePolicies []*InboundTrafficPolicy
}

// IngressTrafficMatch defines the attributes to match ingress traffic for a given backend
type IngressTrafficMatch struct {
	Name                     string
	Port                     uint32
	Protocol                 string
	SourceIPRanges           []string
	ServerNames              []string
	SkipClientCertValidation bool
}

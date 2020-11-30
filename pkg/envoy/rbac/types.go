// Package rbac implements Envoy XDS RBAC policies.
package rbac

// RuleAttribute is the key used for the name of an attribute in a policy Rule
type RuleAttribute string

// Supported attributes for an RBAC principal
const (
	// DownstreamAuthPrincipal is the key used for the name of the downstream principal in a policy Rule
	DownstreamAuthPrincipal RuleAttribute = "downstreamAuthPrincipal"
)

// Rule is a type that can represent a policy's Permission and Principal rules
type Rule struct {
	Attribute RuleAttribute
	Value     string
}

// RulesList is a list of Rule types represented using AND or OR semantics
type RulesList struct {
	AndRules []Rule
	OrRules  []Rule
}

// Policy is a type used to represent an RBAC policy with rules corresponding to Principals and their associated Permissions
type Policy struct {
	Permissions []RulesList
	Principals  []RulesList
}

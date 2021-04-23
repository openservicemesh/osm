package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Egress is the type used to represent an Egress traffic policy.
// An Egress policy allows applications to access endpoints
// external to the service mesh or cluster based on the specified
// rules in the policy.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Egress struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Egress policy specification
	// +optional
	Spec EgressSpec `json:"spec,omitempty"`
}

// EgressSpec is the type used to represent the Egress policy specification
type EgressSpec struct {
	// Sources defines the list of sources the Egress policy is applicable to
	Sources []SourceSpec `json:"sources"`

	// Hosts defines the list of external hosts the Egress policy should allow
	// +optional
	Hosts []string `json:"hosts,omitempty"`

	// IPAddresses defines the list of external IP addresses the Egress policy is applicable to
	// +optional
	IPAddresses []string `json:"ipAddresses,omitempty"`

	// Ports defines the list of ports the Egress policy is applicable to
	Ports []PortSpec `json:"ports"`

	// Matches defines the list of routes the Egress policy should match on
	// +optional
	Matches []corev1.TypedLocalObjectReference `json:"matches,omitempty"`
}

// SourceSpec is the type used to represent the Source in the list of Sources specified in an Egress policy specification
type SourceSpec struct {
	// Kind defines the kind for the source in the Egress policy, ex. ServiceAccount
	Kind string `json:"kind"`

	// Name defines the name of the source for the given Kind
	Name string `json:"name"`

	// Namespace defines the namespace for the given source
	Namespace string `json:"namespace"`
}

// PortSpec is the type used to represent the Port in the list of Ports specified in an Egress policy specification
type PortSpec struct {
	// Number defines the port number
	Number int `json:"number"`

	// Protocol defines the protocol served by the port
	Protocol string `json:"protocol"`
}

// EgressList defines the list of Egress objects
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EgressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Egress `json:"items"`
}

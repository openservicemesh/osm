package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Retry is the type used to represent a Retry policy.
// A Retry policy authorizes retries to failed attempts for outbound traffic
// from one service source to one or more destination services.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Retry struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Retry policy specification
	// +optional
	Spec RetrySpec `json:"spec,omitempty"`
}

// RetrySpec is the type used to represent the Retry policy specification.
type RetrySpec struct {
	// Source defines the source the Retry policy applies to.
	Source RetrySrcDstSpec `json:"source"`

	// Destinations defines the list of destinations the Retry policy applies to.
	Destinations []RetrySrcDstSpec `json:"destinations"`

	// RetryPolicy defines the retry policy the Retry policy applies.
	RetryPolicy RetryPolicySpec `json:"retryPolicy"`
}

// RetrySrcDstSpec is the type used to represent the Destination in the list of Destinations and the Source
// specified in the Retry policy specification.
type RetrySrcDstSpec struct {
	// Kind defines the kind for the Src/Dst in the Retry policy.
	Kind string `json:"kind"`

	// Name defines the name of the Src/Dst for the given Kind.
	Name string `json:"name"`

	// Namespace defines the namespace for the given Src/Dst.
	Namespace string `json:"namespace"`
}

// RetryPolicySpec is the type used to represent the retry policy specified in the Retry policy specification.
type RetryPolicySpec struct {
	// RetryOn defines the policies to retry on, delimited by comma.
	RetryOn string `json:"retryOn"`

	// PerTryTimeout defines the time allowed for a retry before it's considered a failed attempt.
	PerTryTimeout string `json:"perTryTimeout"`

	// NumRetries defines the max number of retries to attempt.
	NumRetries int `json:"numRetries"`

	// RetryBackoffBaseInterval defines the base interval for exponential retry backoff.
	RetryBackoffBaseInterval string `json:"retryBackoffInterval"`
}

// RetryList defines the list of Retry objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Retry `json:"items"`
}

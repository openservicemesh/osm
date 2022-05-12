package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressBackend is the type used to represent an Ingress backend policy.
// An Ingress backend policy authorizes one or more backends to accept
// ingress traffic from one or more sources.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IngressBackend struct {
	// Object's type metadata
	metav1.TypeMeta `json:",inline"`

	// Object's metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the Ingress backend policy specification
	// +optional
	Spec IngressBackendSpec `json:"spec,omitempty"`

	// Status is the status of the IngressBackend configuration.
	// +optional
	Status IngressBackendStatus `json:"status,omitempty"`
}

// IngressBackendSpec is the type used to represent the IngressBackend policy specification.
type IngressBackendSpec struct {
	// Backends defines the list of backends the IngressBackend policy applies to.
	Backends []BackendSpec `json:"backends"`

	// Sources defines the list of sources the IngressBackend policy applies to.
	Sources []IngressSourceSpec `json:"sources"`

	// Matches defines the list of object references the IngressBackend policy should match on.
	// +optional
	Matches []corev1.TypedLocalObjectReference `json:"matches,omitempty"`
}

// BackendSpec is the type used to represent a Backend specified in the IngressBackend policy specification.
type BackendSpec struct {
	// Name defines the name of the backend.
	Name string `json:"name"`

	// Port defines the specification for the backend's port.
	Port PortSpec `json:"port"`

	// TLS defines the specification for the backend's TLS configuration.
	// +optional
	TLS TLSSpec `json:"tls,omitempty"`
}

const (
	// KindService is the kind corresponding to a Service resource.
	KindService = "Service"

	// KindAuthenticatedPrincipal is the kind corresponding to an authenticated principal.
	KindAuthenticatedPrincipal = "AuthenticatedPrincipal"

	// KindIPRange is the kind corresponding to an IP address range represented in CIDR notation.
	KindIPRange = "IPRange"
)

// IngressSourceSpec is the type used to represent the Source in the list of Sources specified in an
// IngressBackend policy specification.
type IngressSourceSpec struct {
	// Kind defines the kind for the source in the IngressBackend policy.
	// Must be one of: Service, AuthenticatedPrincipal, IPRange
	Kind string `json:"kind"`

	// Name defines the name of the source for the given Kind.
	Name string `json:"name"`

	// Namespace defines the namespace for the given source.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// TLSSpec is the type used to represent the backend's TLS configuration.
type TLSSpec struct {
	// SkipClientCertValidation defines whether the backend should skip validating the
	// certificate presented by the client.
	SkipClientCertValidation bool `json:"skipClientCertValidation"`

	// SNIHosts defines the SNI hostnames that the backend allows the client to connect to.
	// +optional
	SNIHosts []string `json:"sniHosts,omitempty"`
}

// IngressBackendList defines the list of IngressBackend objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type IngressBackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []IngressBackend `json:"items"`
}

// IngressBackendStatus is the type used to represent the status of an IngressBackend resource.
type IngressBackendStatus struct {
	// CurrentStatus defines the current status of an IngressBackend resource.
	// +optional
	CurrentStatus string `json:"currentStatus,omitempty"`

	// Reason defines the reason for the current status of an IngressBackend resource.
	// +optional
	Reason string `json:"reason,omitempty"`
}

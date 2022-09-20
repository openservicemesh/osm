package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExtensionService defines the configuration of the external service
// that an OSM managed mesh integrates with.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExtensionService struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the specification of the extension service.
	// +optional
	Spec ExtensionServiceSpec `json:"spec,omitempty"`
}

// ExtensionServiceSpec defines the specification of the extension service.
type ExtensionServiceSpec struct {
	// Host defines the hostname of the extension service.
	Host string `json:"host"`

	// Port defines the port number of the extension service.
	Port uint32 `json:"port"`

	// Protocol defines the protocol of the extension service.
	Protocol string `json:"protocol"`

	// ConnectTimeout defines the timeout for connecting to the extension service.
	// +optional
	ConnectTimeout *metav1.Duration `json:"connectTimeout,omitempty"`
}

// ExtensionServiceList defines the list of ExtensionService objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ExtensionServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ExtensionService `json:"items"`
}

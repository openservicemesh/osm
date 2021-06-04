package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RemoteService is the type used to represent the mesh configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RemoteService struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the RemoteService specification.
	// +optional
	Spec RemoteServiceSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type RemoteServiceSpec struct {
	Cluster ClusterSpec `json:"cluster,omitempty"`
	Equivalence EquivalenceSpec `json:"equivalence,omitempty"`
}

type EquivalenceSpec struct {
	ServiceName string `json:"serviceName,omitempty"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	ServiceNamespace string `json:"serviceNamespace,omitempty"`
}

type ClusterSpec struct {
	Gateway string `json:"Gateway,omitempty"`
	Name string `json:"name,omitempty"`
	Certificate string `json:"certificate,omitempty"`
}

// RemoteServiceList lists the RemoteService objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RemoteServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []RemoteService `json:"items"`
}

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterService is the type used to represent the mesh configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiClusterService struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the MultiClusterService specification.
	// +optional
	Spec MultiClusterServiceSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type MultiClusterServiceSpec struct {
	Cluster        []ClusterSpec `json:"cluster,omitempty"`
	ServiceAccount string        `json:"serviceAccount,omitempty"`
}

type ClusterSpec struct {
	Address     string `json:"address,omitempty"`
	Name        string `json:"name,omitempty"`
	Certificate string `json:"certificate,omitempty"`
}

// MultiClusterServiceList lists the MultiClusterService objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiClusterServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MultiClusterService `json:"items"`
}

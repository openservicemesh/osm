package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiCluster is the type used to represent the multicluster configuration.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiCluster struct {
	// Object's type metadata.
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Object's metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Spec is the MultiClusterService specification.
	// +optional
	Spec MultiClusterSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// MultiClusterSpec is the type used to represent the multicluster specification.
type MultiClusterSpec struct {
	// SourceCluster declares the local cluster where this resource is applied
	SourceCluster ClusterSpec `json:"sourceCluster"`

	// RemoteClusters declares a list of remote clusters which source cluster is aware of
	RemoteClusters []ClusterSpec `json:"remoteClusters"`

	// Services declares a list of services which are deployed on remote clusters.
	Services []MultiClusterServiceSpec `json:"services"`
}

// MultiClusterList defines the list of MultiCluster objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MultiCluster `json:"items"`
}

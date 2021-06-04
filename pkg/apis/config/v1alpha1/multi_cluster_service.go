package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MultiClusterService is the type used to represent the multicluster configuration.
// MultiClusterService name needs to match the name of the service backing the pods in each cluster.
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

// MultiClusterServiceSpec is the type used to represent the multicluster service specification.
type MultiClusterServiceSpec struct {
	// ClusterSpec defines the configuration of other clusters
	Cluster []ClusterSpec `json:"cluster,omitempty"`

	// ServiceAccount represents the service account of the multicluster service.
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// ClusterSpec is the type used to represent a remote cluster in multicluster scenarios.
type ClusterSpec struct {

	// Address defines the remote IP address of the gateway
	Address string `json:"address,omitempty"`

	// Name defines the name of the remote cluster.
	Name string `json:"name,omitempty"`
}

// MultiClusterServiceList defines the list of MultiClusterService objects.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MultiClusterServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MultiClusterService `json:"items"`
}

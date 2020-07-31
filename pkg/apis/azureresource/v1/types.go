package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureResource is an object describing Azure specific compute, which will be included in the NamespacedService Mesh.
type AzureResource struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AzureResourceSpec `json:"spec"`
}

// AzureResourceSpec defines the properties that uniquely identify an Azure compute resource.
type AzureResourceSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ResourceID is the URI identifying an unique Azure compute resource.
	// example: /resource/subscriptions/e3f0/resourceGroups/mesh-rg/providers/Microsoft.Compute/virtualMachineScaleSets/baz
	ResourceID string `json:"resourceid"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AzureResourceList is the list of Azure resources.
type AzureResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AzureResource `json:"items"`
}

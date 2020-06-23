package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSMConfig is an object with configurtanio key/values for the OSM control plane.
type OSMConfig struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OSMConfigSpec `json:"spec"`
}

// OSMConfigSpec defines the properties necessary to configure the OSM control plane.
type OSMConfigSpec struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// LogVerbosity is the verbosity level of the logging system.
	LogVerbosity string `json:"logVerbosity"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OSMConfigList is the list of Azure resources.
type OSMConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OSMConfig `json:"items"`
}

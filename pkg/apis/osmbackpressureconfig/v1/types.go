package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OSMBackpressureConfig is the targets AGIC is allowed to mutate
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OSMBackpressureConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OSMBackpressureConfigSpec `json:"spec"`
}

// OSMBackpressureConfigSpec defines a list of uniquely identifiable targets for which the AGIC is not allowed to mutate config.
type OSMBackpressureConfigSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Max number of requests a connection can make
	MaxRequestsPerConnection int32 `json:"maxrequestsperconnection,omitempty"`
}

// OSMBackpressureConfigList is the list of prohibited targets
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OSMBackpressureConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []OSMBackpressureConfig `json:"items"`
}
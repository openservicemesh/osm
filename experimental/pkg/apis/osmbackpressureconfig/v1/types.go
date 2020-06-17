package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OSMBackpressureConfig is ...
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OSMBackpressureConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OSMBackpressureConfigSpec `json:"spec"`
}

// OSMBackpressureConfigSpec is ...
type OSMBackpressureConfigSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// MaxRequestsPerConnection is the max number of requests a connection can make
	MaxRequestsPerConnection int32 `json:"maxrequestsperconnection,omitempty"`
}

// OSMBackpressureConfigList is ...
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type OSMBackpressureConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of OSMBackpressureConfig
	Items []OSMBackpressureConfig `json:"items"`
}

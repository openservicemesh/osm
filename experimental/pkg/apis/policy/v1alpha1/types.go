package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Backpressure is ...
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Backpressure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BackpressureSpec `json:"spec"`
}

// BackpressureSpec is ...
type BackpressureSpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// MaxRequestsPerConnection is the max number of requests a connection can make
	MaxRequestsPerConnection int32 `json:"maxrequestsperconnection,omitempty"`
}

// BackpressureList is ...
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type BackpressureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of Backpressure
	Items []Backpressure `json:"items"`
}
